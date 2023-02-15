package handoff

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Server struct {
	port int

	// immutable readonly map of testsuites
	testSuites map[string]TestSuite
	testRuns   sync.Map

	// global runID counter, this should be per TestSuite in the future
	currentRun int32

	events chan Event
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
	go s.eventLoop()

	return s.runHTTP()
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

		s.testRuns.Store(key, e.Apply(testRun))

		switch e := e.(type) {
		case TestRunStarted:
			ts := s.testSuites[e.suiteName]
			go s.runTests(ts, e.runID)
		}
	}
}

func (s *Server) runTests(suite TestSuite, runID int32) {
	start := time.Now()

	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			log.Printf("setup of suite %s failed: %v\n", suite.Name, err)

			s.events <- TestRunSetupFailed{
				TestRunIdentifier: TestRunIdentifier{
					runID:     runID,
					suiteName: suite.Name,
				},
				start: start, end: time.Now(), err: err}
			return
		}
	}

	for name, test := range suite.Tests {
		s.executeTest(suite, runID, name, test)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			log.Printf("teardown of suite %s failed: %v\n", suite.Name, err)
		}
	}

	s.events <- TestRunFinished{
		TestRunIdentifier: TestRunIdentifier{
			runID:     runID,
			suiteName: suite.Name,
		},
		start: start, end: time.Now()}
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

func testRunKey(suiteName string, runID int32) string {
	return fmt.Sprintf("%s-%d", suiteName, runID)
}
