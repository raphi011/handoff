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

	plugins plugins
	// immutable readonly map of testsuites
	testSuites map[string]TestSuite
	testRuns   sync.Map
	schedules  []ScheduledRun
	cron       *cron.Cron

	frontendPort int

	// global runID counter, this should be per TestSuite in the future
	currentRun int32

	events chan Event
}

type plugins struct {
	all               []Plugin
	testStarted       []TestStartedListener
	testFinished      []TestFinishedListener
	testSuiteFinished []TestSuiteFinishedListener
	testSuiteStarted  []TestSuiteStartedListener
}

type Option func(s *Server)

func New(opts ...Option) *Server {
	s := &Server{
		port:         1337,
		frontendPort: 8080,
		testSuites:   map[string]TestSuite{},
		plugins: plugins{
			all:               []Plugin{},
			testStarted:       []TestStartedListener{},
			testFinished:      []TestFinishedListener{},
			testSuiteFinished: []TestSuiteFinishedListener{},
			testSuiteStarted:  []TestSuiteStartedListener{},
		},
		testRuns: sync.Map{},
		events:   make(chan Event, 100),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Server) Start() error {
	if err := s.initPlugins(); err != nil {
		return err
	}
	if err := s.startSchedules(); err != nil {
		return err
	}

	go s.eventLoop()

	return s.runHTTP()
}

func (s *Server) initPlugins() error {
	for _, p := range s.plugins.all {
		if err := p.Init(); err != nil {
			return err
		}

		registeredHook := false

		if l, ok := p.(TestStartedListener); ok {
			s.plugins.testStarted = append(s.plugins.testStarted, l)
			registeredHook = true
		}
		if l, ok := p.(TestFinishedListener); ok {
			s.plugins.testFinished = append(s.plugins.testFinished, l)
			registeredHook = true
		}
		if l, ok := p.(TestSuiteStartedListener); ok {
			s.plugins.testSuiteStarted = append(s.plugins.testSuiteStarted, l)
			registeredHook = true
		}
		if l, ok := p.(TestSuiteFinishedListener); ok {
			s.plugins.testSuiteFinished = append(s.plugins.testSuiteFinished, l)
			registeredHook = true
		}

		if !registeredHook {
			return fmt.Errorf("plugin %s does not implement any hook", p.Name())
		}
	}

	return nil
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
			s.events <- TestRunStartedEvent{
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

		if _, ok := e.(TestRunStartedEvent); !ok {
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
		case TestRunStartedEvent:
			ts := s.testSuites[e.suiteName]
			go s.runTestSuite(ts, testRun)
		}
	}
}

func (s *Server) runTestSuite(suite TestSuite, testRun TestRun) {
	start := time.Now()

	// testSuitesRunMetric.WithLabelValues(suite.AssociatedService, suite.Name).Inc()
	testSuitesRunning := testSuitesRunningMetric.WithLabelValues(suite.AssociatedService, suite.Name)

	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			log.Printf("setup of suite %s failed: %v\n", suite.Name, err)

			s.events <- TestRunSetupFailedEvent{
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
		s.executeTest(suite, testRun, name, test)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			log.Printf("teardown of suite %s failed: %v\n", suite.Name, err)
		}
	}

	testSuitesRunMetric.WithLabelValues(suite.AssociatedService, suite.Name, "PASSED").Inc()

	s.events <- TestRunFinishedEvent{
		TestRunIdentifier: TestRunIdentifier{
			runID:     testRun.ID,
			suiteName: suite.Name,
		},
		start:   start,
		end:     time.Now(),
		skipped: skipped,
	}
}

func (s *Server) executeTest(suite TestSuite, testRun TestRun, name string, test TestFunc) {
	start := time.Now()

	t := T{
		name:       name,
		logs:       []string{},
		passed:     true,
		runContext: map[string]any{},
	}

	s.notifyTestStarted(suite, testRun, name)

	defer func() {
		end := time.Now()

		err := recover()

		testRunsMetric.WithLabelValues(suite.AssociatedService, suite.Name, string(t.Result()))

		s.events <- TestFinishedEvent{
			TestRunIdentifier: TestRunIdentifier{
				runID:     testRun.ID,
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

		s.notifyTestFinished(suite, testRun, name, t.runContext)
	}()

	test(&t)
}

func (s *Server) notifyTestStarted(suite TestSuite, testRun TestRun, name string) {
	for _, p := range s.plugins.testStarted {
		p.TestStarted(suite, testRun, name)
	}
}

func (s *Server) notifyTestFinished(suite TestSuite, testRun TestRun, name string, runContext map[string]any) {
	for _, p := range s.plugins.testFinished {
		p.TestFinished(suite, testRun, name, runContext)
	}
}

func WithPlugin(p Plugin) Option {
	return func(s *Server) {
		s.plugins.all = append(s.plugins.all, p)
	}
}

func WithTestSuite(suite TestSuite) Option {
	return func(s *Server) {
		s.testSuites[suite.Name] = suite
	}
}

func WithFrontendPort(port int) Option {
	return func(s *Server) {
		s.frontendPort = port
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
