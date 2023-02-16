package handoff

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Server struct {
	port int

	plugins Plugins

	// immutable readonly map of testsuites
	testSuites map[string]TestSuite
	testRuns   sync.Map
	schedules  []ScheduledRun
	cron       *cron.Cron

	// global runID counter, this should be per TestSuite in the future
	currentRun int32

	events chan Event
}

type Plugins struct {
	pagerDuty PagerDutyPlugin
	github    GithubPlugin
	slack     SlackPlugin
	logstash  LogstashPlugin
}

type Option func(s *Server)

func New(opts ...Option) *Server {
	s := &Server{
		port:       1337,
		testSuites: map[string]TestSuite{},
		testRuns:   sync.Map{},
		events:     make(chan Event, 100),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Server) Start() error {
	if err := s.startSchedules(); err != nil {
		return err
	}

	go s.eventLoop()

	return s.runHTTP()
}

func (s *Server) startSchedules() error {
	s.cron = cron.New(cron.WithSeconds())

	for i := range s.schedules {
		schedule := s.schedules[i]

		ts, ok := s.testSuites[schedule.TestSuiteName]
		if !ok {
			return fmt.Errorf("test suite %s does not exist", schedule.TestSuiteName)
		}

		entryID, err := s.cron.AddFunc(schedule.Schedule, func() {
			s.events <- TestRunStarted{
				TestRunIdentifier: TestRunIdentifier{runID: s.nextID(), suiteName: schedule.TestSuiteName},
				Scheduled:         time.Now(),
				TriggeredBy:       "scheduled",
				Tests:             len(ts.Tests),
			}
		})

		if err != nil {
			return err
		}

		s.schedules[i].EntryID = entryID
	}

	s.cron.Start()

	return nil
}

// eventLoop loops over all events and updates the testRuns map accordingly.
// It should be started as a goroutine once. The `testRuns` map must only be
// written to from here.
func (s *Server) eventLoop() {
	for e := range s.events {
		key := testRunKey(e.SuiteName(), e.RunID())

		testRun := TestRun{}

		if _, ok := e.(TestRunStarted); !ok {
			val, found := s.testRuns.Load(key)
			if !found {
				log.Printf("could not handle event, run %s not found\n", key)
				return
			}

			testRun = val.(TestRun)
		}

		testRun = e.Apply(testRun)

		s.testRuns.Store(key, testRun)

		switch e := e.(type) {
		case TestRunStarted:
			ts := s.testSuites[e.suiteName]
			go s.runTestSuite(ts, testRun)
		}
	}
}

func (s *Server) runTestSuite(suite TestSuite, testRun TestRun) {
	start := time.Now()

	testSuitesRunMetric.WithLabelValues(suite.AssociatedService, suite.Name).Inc()
	testSuitesRunning := testSuitesRunningMetric.WithLabelValues(suite.AssociatedService, suite.Name)

	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			log.Printf("setup of suite %s failed: %v\n", suite.Name, err)

			s.events <- TestRunSetupFailed{
				TestRunIdentifier: TestRunIdentifier{
					runID:     testRun.ID,
					suiteName: suite.Name,
				},
				start: start, end: time.Now(), err: err}
			return
		}
	}

	skipped := 0

	filter := testRun.testFilterRegex

	for name, test := range suite.Tests {
		if filter != nil && !filter.MatchString(name) {
			skipped++
			continue
		}
		s.executeTest(suite, testRun.ID, name, test)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			log.Printf("teardown of suite %s failed: %v\n", suite.Name, err)
		}
	}

	s.events <- TestRunFinished{
		TestRunIdentifier: TestRunIdentifier{
			runID:     testRun.ID,
			suiteName: suite.Name,
		},
		start:   start,
		end:     time.Now(),
		skipped: skipped,
	}
}

func (s *Server) executeTest(suite TestSuite, runID int32, name string, test TestFunc) {
	start := time.Now()

	t := T{
		name:   name,
		logs:   []string{},
		passed: true,
	}

	defer func() {
		end := time.Now()

		err := recover()

		s.events <- TestFinished{
			TestRunIdentifier: TestRunIdentifier{
				runID:     runID,
				suiteName: suite.Name,
			},
			testName: name,
			recovery: err,
			passed:   t.passed,
			skipped:  t.skipped,
			logs:     t.logs,
			start:    start,
			end:      end,
		}
	}()

	test(&t)
}

func WithTestSuite(suite TestSuite) Option {
	return func(s *Server) {
		s.testSuites[suite.Name] = suite
	}
}

func WithScheduledRun(name, schedule string) Option {
	return func(s *Server) {
		s.schedules = append(s.schedules, ScheduledRun{TestSuiteName: name, Schedule: schedule})
	}
}

func testRunKey(suiteName string, runID int32) string {
	return fmt.Sprintf("%s-%d", suiteName, runID)
}
