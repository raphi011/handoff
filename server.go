package handoff

import (
	"fmt"
	"log"
	"sync"
)

type Server struct {
	port int

	// immutable readonly map of testsuites
	testSuites map[string]TestSuite
	testRuns   sync.Map

	// global runID counter, this should be per TestSuite later
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
	for name, test := range suite.Tests {
		s.executeTest(suite, runID, name, test)
	}
}

func (s *Server) executeTest(suite TestSuite, runID int32, name string, test TestFunc) {
	t := T{
		name:   name,
		logs:   []string{},
		passed: true,
	}

	defer func() {
		err := recover()

		s.events <- TestFinished{
			TestRunIdentifier: TestRunIdentifier{
				runID:     runID,
				suiteName: suite.Name,
			},
			testName: name,
			recovery: err,
			passed:   t.passed,
			logs:     t.logs,
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
