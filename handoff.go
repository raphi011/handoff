package handoff

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/raphi011/handoff/internal/metric"
	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/raphi011/handoff/plugin"
	"github.com/robfig/cron/v3"
	"golang.org/x/exp/slog"
)

type Handoff struct {
	port       int
	serverMode bool

	plugins plugins

	readOnlyTestSuites map[string]model.TestSuite

	schedules []scheduledRun
	cron      *cron.Cron

	storage *storage.Storage

	events chan event
}

// TestSuite represents the external view of the Testsuite to allow users of the library
// to omit passing in redundant information like the name of the test which can be retrieved
// via reflection..
// It is only used by the caller of the library and then mapped internally to enrich the
// struct with e.g. test function names.
type TestSuite struct {
	// Name of the testsuite
	Name string `json:"name"`
	// AssociatedService can optionally contain the name of the service
	// that this testsuite is associated with. This will be used for metric labels.
	AssociatedService string `json:"associatedService"`
	Setup             func() error
	Teardown          func() error
	Tests             []TestFunc
}

type Plugin interface {
	Name() string
	Init() error
	Stop() error
}

// Reexport to allow library users to reference these types

type TestFunc = model.TestFunc
type TB = model.TB

type plugins struct {
	all               []Plugin
	testStarted       []plugin.TestStartedListener
	testFinished      []plugin.TestFinishedListener
	testSuiteFinished []plugin.TestSuiteFinishedListener
	testSuiteStarted  []plugin.TestSuiteStartedListener
}

type option func(s *Handoff)

// New configures a new Handoff instance.
func New(opts ...option) *Handoff {
	s := &Handoff{
		port:               1337,
		serverMode:         false,
		readOnlyTestSuites: map[string]model.TestSuite{},
		plugins: plugins{
			all:               []Plugin{},
			testStarted:       []plugin.TestStartedListener{},
			testFinished:      []plugin.TestFinishedListener{},
			testSuiteFinished: []plugin.TestSuiteFinishedListener{},
			testSuiteStarted:  []plugin.TestSuiteStartedListener{},
		},
		events: make(chan event, 100),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Handoff) Run() error {
	s.parseFlags()

	if err := s.initPlugins(); err != nil {
		return fmt.Errorf("init plugins: %w", err)
	}

	if s.serverMode {
		db, err := storage.New("")
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		s.storage = db

		if err := s.startSchedules(); err != nil {
			return fmt.Errorf("start schedules: %w", err)
		}

		go s.eventLoop()

		return s.runHTTP()
	}

	fmt.Println("CLI Mode, WIP")

	return nil
}

func (s *Handoff) parseFlags() {
	var port = flag.Int("p", s.port, "port used by the server (server mode only)")
	var serverMode = flag.Bool("s", s.serverMode, "enable server mode")
	var listTestSuites = flag.Bool("l", false, "list all configured test suites and exit")

	flag.Parse()

	if *listTestSuites {
		s.printTestSuites()
	}

	s.port = *port
	s.serverMode = *serverMode
}

func (s *Handoff) printTestSuites() {
	b := strings.Builder{}

	for _, ts := range s.readOnlyTestSuites {
		b.WriteString("suite: \"" + ts.Name + "\"")
		if ts.AssociatedService != "" {
			b.WriteString(" (" + ts.AssociatedService + ")")
		}
		b.WriteString("\n")

		for _, t := range ts.Tests {
			b.WriteString("\t " + t.Name + "\n")
		}
	}

	fmt.Print(b.String())

	os.Exit(0)
}

func (s *Handoff) initPlugins() error {
	for _, p := range s.plugins.all {
		if err := p.Init(); err != nil {
			return fmt.Errorf("initiating plugin %q: %w", p.Name(), err)
		}

		registeredHook := false

		if l, ok := p.(plugin.TestStartedListener); ok {
			s.plugins.testStarted = append(s.plugins.testStarted, l)
			registeredHook = true
		}
		if l, ok := p.(plugin.TestFinishedListener); ok {
			s.plugins.testFinished = append(s.plugins.testFinished, l)
			registeredHook = true
		}
		if l, ok := p.(plugin.TestSuiteStartedListener); ok {
			s.plugins.testSuiteStarted = append(s.plugins.testSuiteStarted, l)
			registeredHook = true
		}
		if l, ok := p.(plugin.TestSuiteFinishedListener); ok {
			s.plugins.testSuiteFinished = append(s.plugins.testSuiteFinished, l)
			registeredHook = true
		}

		if !registeredHook {
			return fmt.Errorf("plugin %q does not implement any hook", p.Name())
		}
	}

	return nil
}

func (s *Handoff) startSchedules() error {
	s.cron = cron.New(cron.WithSeconds())

	for i := range s.schedules {
		schedule := s.schedules[i]

		ts, ok := s.readOnlyTestSuites[schedule.TestSuiteName]
		if !ok {
			return fmt.Errorf("starting scheduled test suite run: test suite %q not found", schedule.TestSuiteName)
		}

		entryID, err := s.cron.AddFunc(schedule.Schedule, func() {
			s.events <- testRunStartedEvent{
				testRunIdentifier: testRunIdentifier{suiteName: schedule.TestSuiteName},
				scheduled:         time.Now(),
				triggeredBy:       "scheduled",
				tests:             len(ts.Tests),
			}
		})

		if err != nil {
			return fmt.Errorf("adding scheduled test suite run %q: %w", schedule.TestSuiteName, err)
		}

		s.schedules[i].EntryID = entryID
	}

	s.cron.Start()

	return nil
}

// eventLoop loops over all events and updates the testRuns map accordingly.
// It should be started as a goroutine once. The `testRuns` map must only be
// written to from here.
func (s *Handoff) eventLoop() {
	for e := range s.events {
		ctx := context.Background()

		testSuiteRun := model.TestSuiteRun{}

		isNewTestRun := e.RunID() < 1

		if !isNewTestRun {
			var err error

			testSuiteRun, err = s.storage.LoadTestSuiteRun(ctx, e.SuiteName(), e.RunID())
			if err != nil {
				slog.Error("could not handle event", "error", err, "run-id", e.RunID(), "event", fmt.Sprintf("%T", e))
				continue
			}
		}

		testSuiteRun = e.Apply(testSuiteRun)

		var err error

		if isNewTestRun {
			testSuiteRun.ID, err = s.storage.SaveTestSuiteRun(ctx, testSuiteRun)
		} else {
			err = s.storage.UpdateTestSuiteRun(ctx, testSuiteRun)
		}

		// `TestResults` will only contain new / updated testresults so we can assume
		// that every entry needs to be persisted.
		for _, tr := range testSuiteRun.TestResults {
			s.storage.UpsertTestRun(ctx, testSuiteRun.ID, tr)
		}

		if err != nil {
			slog.Error("could not persist test suite run", "error", err)
			continue
		}

		switch e := e.(type) {
		case testRunStartedEvent:
			ts := s.readOnlyTestSuites[e.suiteName]

			go s.runTestSuite(ts, testSuiteRun)
		}
	}
}

func (s *Handoff) runTestSuite(suite model.TestSuite, testRun model.TestSuiteRun) {
	start := time.Now()

	testSuitesRunning := metric.TestSuitesRunning.WithLabelValues(suite.AssociatedService, suite.Name)

	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			slog.Error("setup of suite failed", "suite-name", suite.Name, "error", err)

			s.events <- testRunSetupFailedEvent{
				testRunIdentifier: testRunIdentifier{
					runID:     testRun.ID,
					suiteName: suite.Name,
				},
				start: start, end: time.Now(), err: err}
			return
		}
	}

	skipped := 0

	filter := testRun.TestFilterRegex

	for _, test := range suite.Tests {
		if filter != nil && !filter.MatchString(test.Name) {
			skipped++
			continue
		}
		s.executeTest(suite, testRun, test.Name, test.Func)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			slog.Warn("teardown of suite failed", "suite-name", suite.Name, "error", err)
		}
	}

	metric.TestSuitesRun.WithLabelValues(suite.AssociatedService, suite.Name, "PASSED").Inc()

	s.events <- testRunFinishedEvent{
		testRunIdentifier: testRunIdentifier{
			runID:     testRun.ID,
			suiteName: suite.Name,
		},
		start:   start,
		end:     time.Now(),
		skipped: skipped,
	}
}

func (s *Handoff) executeTest(suite model.TestSuite, testRun model.TestSuiteRun, name string, test model.TestFunc) {
	start := time.Now()

	t := t{
		name:       name,
		passed:     true,
		runContext: map[string]any{},
	}

	s.notifyTestStarted(suite, testRun, name)

	defer func() {
		end := time.Now()

		err := recover()

		metric.TestRunsTotal.WithLabelValues(suite.AssociatedService, suite.Name, string(t.Result()))

		s.events <- testFinishedEvent{
			testRunIdentifier: testRunIdentifier{
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

func (s *Handoff) notifyTestStarted(suite model.TestSuite, testRun model.TestSuiteRun, name string) {
	for _, p := range s.plugins.testStarted {
		p.TestStarted(suite, testRun, name)
	}
}

func (s *Handoff) notifyTestFinished(suite model.TestSuite, testRun model.TestSuiteRun, name string, runContext map[string]any) {
	for _, p := range s.plugins.testFinished {
		p.TestFinished(suite, testRun, name, runContext)
	}
}
