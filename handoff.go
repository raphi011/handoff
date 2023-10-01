package handoff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"plugin"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/raphi011/handoff/internal/metric"
	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/robfig/cron/v3"
	"golang.org/x/exp/slog"
)

type Handoff struct {
	// server configuration
	config config

	// configured plugins
	plugins *pluginManager

	// a map of all testsuites that must not be modified after
	// initialisation.
	readOnlyTestSuites map[string]model.TestSuite

	// scheduled runs configured by the user
	schedules []ScheduledRun

	// cron object used for scheduled runs
	cron *cron.Cron

	runningTestSuites sync.WaitGroup

	httpServer *http.Server

	storage *storage.Storage
}

type config struct {
	// Port for the web api
	Port int `arg:"-p,env:HANDOFF_PORT" help:"port used by the server (server mode only)" default:"1337"`

	// location of the sqlite database file, if empty we default
	// to an in-memory database.
	DatabaseFilePath string `arg:"-d,--database,env:HANDOFF_DATABASE" help:"database file location" default:"handoff.db"`

	// TestSuiteLibraryFiles is a list of library files that will load
	// additional test suites and scheduled runs.
	TestSuiteLibraryFiles []string `arg:"-t,--testsuite,env:HANDOFF_TESTSUITE_FILE" help:"optional list of test suite library files"`

	// List will, if set to true, print all loaded test suites
	// and immediately exit.
	ListTestSyutes bool `arg:"-l,--list" help:"list all configured test suites and exit" default:"false"`

	// Environment is e.g. the cluster/platform the tests are run on.
	// This is added to metrics and the testrun information.
	Environment string `arg:"-e,--env,env:HANDOFF_ENVIRONMENT" help:"the environment where the tests are run"`
}

func (c config) Version() string {
	// TODO: use debug/buildinfo to fetch git info?
	return "Handoff (alpha)"
}

// TestSuite represents the external view of the Testsuite to allow users of the library
// to omit passing in redundant information like the name of the test which can be retrieved
// via reflection..
// It is only used by the caller of the library and then mapped internally to enrich the
// struct with e.g. test function names.
type TestSuite struct {
	// Name of the testsuite
	Name string `json:"name"`
	// Namespace allows grouping of test suites, e.g. by team name.
	Namespace string
	Setup     func() error
	Teardown  func() error
	Tests     []TestFunc
}

// Reexport to allow library users to reference these types
type TestFunc = model.TestFunc
type TB = model.TB
type TestContext = model.TestContext

type option func(s *Handoff)

// New configures a new Handoff instance.
func New(opts ...option) *Handoff {
	h := &Handoff{
		config: config{
			Port: 1337,
		},
		readOnlyTestSuites: map[string]model.TestSuite{},
	}

	h.plugins = newPluginManager(h.asyncPluginCallback)

	for _, o := range opts {
		o(h)
	}

	return h
}

func (h *Handoff) Run() error {
	arg.MustParse(&h.config)

	if err := h.loadLibraryFiles(); err != nil {
		return fmt.Errorf("loading test library files: %w", err)
	}

	if h.config.ListTestSyutes {
		h.printTestSuites()
	}

	if err := h.plugins.init(); err != nil {
		return fmt.Errorf("init plugins: %w", err)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	db, err := storage.New(h.config.DatabaseFilePath)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	h.storage = db

	if err := h.startSchedules(); err != nil {
		return fmt.Errorf("start schedules: %w", err)
	}

	go h.runHTTP()

	h.restartPendingTestRuns()

	signal := <-exit

	h.gracefulShutdown(signal)

	return nil
}

func (h *Handoff) loadLibraryFiles() error {
	for _, f := range h.config.TestSuiteLibraryFiles {
		p, err := plugin.Open(f)
		if err != nil {
			return fmt.Errorf("opening test suite file: %w", err)
		}

		handoffSymbol, err := p.Lookup("Handoff")
		if err != nil {
			return fmt.Errorf("invalid test suite file: %w", err)
		}

		handoffFunc, ok := handoffSymbol.(func() ([]TestSuite, []ScheduledRun))
		if !ok {
			return errors.New("invalid test suite plugin, expected signature `func TestSuites() []TestSuite``")
		}

		suites, schedules := handoffFunc()

		for _, suite := range suites {
			h.readOnlyTestSuites[suite.Name] = mapTestSuite(suite)
		}

		h.schedules = append(h.schedules, schedules...)
	}

	return nil
}

func (s *Handoff) restartPendingTestRuns() {
	// pendingRuns, err := s.storage.LoadPendingTestSuiteRuns(context.Background())
	// if err != nil {
	// 	slog.Warn("Unable to load pending test suite runs", "error", err)
	// 	return
	// }
}

func (s *Handoff) gracefulShutdown(signal os.Signal) {
	slog.Info("Received signal, shutting down", "signal", signal.String())

	httpStopCtx := s.stopHTTP()
	cronStopCtx := s.cron.Stop()

	<-httpStopCtx.Done()
	slog.Info("Http listener stopped")
	<-cronStopCtx.Done()
	slog.Info("Scheduled tests stopped")

	pluginStopCtx := s.plugins.shutdown()
	<-pluginStopCtx.Done()
	slog.Info("Plugins stopped")

	s.runningTestSuites.Wait()

	if err := s.storage.Close(); err != nil {
		slog.Warn("Closing DB connection failed", "error", err)
		return
	}
	slog.Info("DB connection closed")
	slog.Info("Shutdown successful")
}

func (s *Handoff) printTestSuites() {
	b := strings.Builder{}

	for _, ts := range s.readOnlyTestSuites {
		b.WriteString("suite: \"" + ts.Name + "\"")
		if ts.Namespace != "" {
			b.WriteString(" (" + ts.Namespace + ")")
		}
		b.WriteString("\n")

		for name := range ts.Tests {
			b.WriteString("\t " + name + "\n")
		}
	}

	fmt.Print(b.String())

	os.Exit(0)
}

func (s *Handoff) startSchedules() error {
	s.cron = cron.New(cron.WithSeconds())

	for i := range s.schedules {
		schedule := s.schedules[i]

		if schedule.TestFilter != "" {
			var err error
			schedule.testFilter, err = regexp.Compile(schedule.TestFilter)
			if err != nil {
				return fmt.Errorf("invalid filter regex %s: %w", schedule.TestFilter, err)
			}
		}

		ts, ok := s.readOnlyTestSuites[schedule.TestSuiteName]
		if !ok {
			return fmt.Errorf("test suite %q not found", schedule.TestSuiteName)
		}

		if len(ts.FilterTests(schedule.testFilter)) == 0 {
			return errors.New("no tests match filter regex %s")
		}

		entryID, err := s.cron.AddFunc(schedule.Schedule, func() {
			_, err := s.startNewTestSuiteRun(ts, "scheduled", schedule.testFilter)
			if err != nil {
				slog.Error("starting new scheduled test suite run failed", "error", err, "test-suite", ts.Name)
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

// startNewTestSuiteRun is used to start new test suite runs. It persists the
// test suite run in a pending state and kicks off the execution of the run in a separate
// goroutine.
func (s *Handoff) startNewTestSuiteRun(ts model.TestSuite, triggeredBy string, testFilter *regexp.Regexp) (model.TestSuiteRun, error) {
	testFilterString := ""
	if testFilter != nil {
		testFilterString = testFilter.String()
	}
	tsr := model.TestSuiteRun{
		SuiteName:       ts.Name,
		TestResults:     []model.TestRun{},
		Result:          model.ResultPending,
		TestFilter:      testFilterString,
		TestFilterRegex: testFilter,
		Tests:           len(ts.Tests),
		Scheduled:       time.Now(),
		TriggeredBy:     triggeredBy,
		Environment:     s.config.Environment,
	}

	ctx, err := s.storage.StartTransaction(context.Background())
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to start transaction: %w", err)
	}
	defer s.storage.RollbackTransaction(ctx)

	tsr.ID, err = s.storage.SaveTestSuiteRun(ctx, tsr)
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to persist new test suite run: %w", err)
	}

	for testName := range ts.Tests {
		result := model.ResultPending

		if tsr.TestFilterRegex != nil && !tsr.TestFilterRegex.MatchString(testName) {
			result = model.ResultSkipped
		}

		tr := model.TestRun{
			SuiteName:  tsr.SuiteName,
			SuiteRunID: tsr.ID,
			Name:       testName,
			Result:     result,
			Attempt:    1,
			Forced:     false,
			Context:    model.TestContext{},
		}

		err = s.storage.InsertTestRun(ctx, tr)
		if err != nil {
			return model.TestSuiteRun{}, fmt.Errorf("unable to persist new test run: %w", err)
		}

		tsr.TestResults = append(tsr.TestResults, tr)
	}

	if err = s.storage.CommitTransaction(ctx); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to persist the test suite run: %w", err)
	}

	go s.runTestSuite(ts, tsr, tsr.TestFilterRegex, false)

	return tsr, nil
}

// runTestSuite executes a test suite run. The test suite run can be new or one that
// is continued/rerun. The testFilter is used to run a subset of tests. If nil the testFilter
// of the TestSuiteRun is used (if any). If `forceRun` is set to true all tests that match the
// filter are executed again even if a previous attempt succeeded or failed.
func (s *Handoff) runTestSuite(
	suite model.TestSuite,
	tsr model.TestSuiteRun,
	testFilter *regexp.Regexp,
	forceRun bool,
) {
	s.runningTestSuites.Add(1)
	defer s.runningTestSuites.Done()

	ctx := context.Background()

	timeNotSet := time.Time{}
	if tsr.Start == timeNotSet {
		tsr.Start = time.Now()
		if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
			slog.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
		}
	}

	testSuitesRunning := metric.TestSuitesRunning.WithLabelValues(suite.Namespace, suite.Name)
	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			slog.Warn("setup of suite failed", "suite-name", suite.Name, "error", err)

			tsr.Result = model.ResultSetupFailed
			tsr.End = time.Now()
			if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
				slog.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
			}
			return
		}
	}

	latestTestAttempt := func(testName string) *model.TestRun {
		testResult := &model.TestRun{}
		for i := range tsr.TestResults {
			t := &tsr.TestResults[i]
			if t.Name == testName && t.Attempt > testResult.Attempt {
				testResult = t
			}
		}

		return testResult
	}

	if testFilter == nil {
		testFilter = tsr.TestFilterRegex
	}

	for testName := range suite.FilterTests(testFilter) {
		// todo: check if the server is shutting down and if so return early.
		// we will continue pending test runs when the server starts up again.

		tr := latestTestAttempt(testName)

		if forceRun {
			forcedRun := tr.NewForcedAttempt()

			var err error
			forcedRun.Attempt, err = s.storage.InsertForcedTestRun(ctx, forcedRun)
			if err != nil {
				slog.Error("unable to persist forced test run", "error", err)
				return
			}

			tsr.TestResults = append(tsr.TestResults, forcedRun)

			tr = &tsr.TestResults[len(tsr.TestResults)-1]

		} else if tr.Result != model.ResultPending {
			continue
		}

		s.runTest(suite, &tsr, tr)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			slog.Warn("teardown of suite failed", "suite-name", suite.Name, "error", err)
		}
	}

	metric.TestSuitesRun.WithLabelValues(suite.Namespace, suite.Name, "PASSED").Inc()

	// TODO: Add the plugin context to the `testRunFinishedEvent`
	s.plugins.notifyTestSuiteFinished(suite, tsr)

	tsr.End = time.Now()
	tsr.Result = tsr.ResultFromTestResults()
	tsr.DurationInMS = tsr.TestSuiteDuration()

	if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
		slog.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
	}

	s.plugins.notifyTestSuiteFinishedAsync(suite, tsr)
}

// runTest runs an individual test that is part of a test suite. This function must only be called
// by `runTestSuite()`.
func (s *Handoff) runTest(suite model.TestSuite, testSuiteRun *model.TestSuiteRun, testRun *model.TestRun) {
	t := T{
		suiteName:      suite.Name,
		testName:       testRun.Name,
		runtimeContext: map[string]any{},
	}

	start := time.Now()

	defer func() {
		end := time.Now()

		err := recover()

		result := t.Result()

		metric.TestRunsTotal.WithLabelValues(suite.Namespace, suite.Name, string(result)).Inc()

		s.plugins.notifyTestFinished(suite, *testSuiteRun, testRun.Name, t.runtimeContext)

		logs := t.logs

		if err != nil && t.result != model.ResultSkipped {
			if _, ok := err.(failTestErr); !ok {
				// this is an unexpected panic (does not originate from handoff)
				logs.WriteString(fmt.Sprintf("%v\n", err))
				result = model.ResultFailed
			}
		}

		testRun.Start = start
		testRun.End = end
		testRun.DurationInMS = end.Sub(start).Milliseconds()
		testRun.Result = result
		testRun.SoftFailure = t.softFailure
		testRun.Logs = logs.String()
		testRun.Context = t.runtimeContext

		if err := s.storage.UpdateTestRun(context.Background(), *testRun); err != nil {
			slog.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
		}

		s.plugins.notifyTestFinishedAync(suite, *testSuiteRun, testRun.Name, t.runtimeContext)

		t.runTestCleanup()
	}()

	suite.Tests[testRun.Name](&t)
}

// asyncPluginCallback is called by asynchronous plugin hooks and persists the updated plugincontext change.
func (s *Handoff) asyncPluginCallback(p Plugin, pluginContext map[string]any) {
	// todo
}
