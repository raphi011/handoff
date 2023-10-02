package handoff

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"plugin"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/raphi011/handoff/internal/metric"
	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/robfig/cron/v3"
)

type Server struct {
	// server configuration
	config config

	// configured plugins
	plugins *pluginManager

	// a map of all testsuites that must not be modified after
	// initialisation.
	readOnlyTestSuites map[string]model.TestSuite

	// _userProvidedTestSuites is a list of all test suites provided
	// by the user and will be mapped to `readOnlyTestSuites` on startup.
	_userProvidedTestSuites []TestSuite

	// scheduled runs configured by the user
	schedules []ScheduledRun

	shuttingDown bool

	// started will be closed when the service has started.
	started chan any
	// shutdown will be closed when the service has shut down.
	shutdown chan any

	// exit receives os signals
	exit chan os.Signal

	// cron object used for scheduled runs
	cron *cron.Cron

	runningTestSuites sync.WaitGroup

	httpServer *http.Server

	storage *storage.Storage
}

type config struct {
	HostIP string `arg:"-h,--host,env:HANDOFF_HOST" help:"ip address the server should bind to" default:"localhost"`

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
	ListTestSuites bool `arg:"-l,--list" help:"list all configured test suites and exit" default:"false"`

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

type option func(s *Server)

// New configures a new Handoff instance.
func New(opts ...option) *Server {
	h := &Server{
		readOnlyTestSuites: map[string]model.TestSuite{},
		started:            make(chan any),
		exit:               make(chan os.Signal, 1),
		shutdown:           make(chan any),
	}

	h.plugins = newPluginManager(h.asyncPluginCallback)

	for _, o := range opts {
		o(h)
	}

	return h
}

func (h *Server) Run() error {
	startupStart := time.Now()
	arg.MustParse(&h.config)

	if err := h.loadLibraryFiles(); err != nil {
		return fmt.Errorf("loading test library files: %w", err)
	}

	if err := h.mapTestSuites(); err != nil {
		return err
	}

	if h.config.ListTestSuites {
		h.printTestSuites()
	}

	if err := h.plugins.init(); err != nil {
		return fmt.Errorf("init plugins: %w", err)
	}

	signal.Notify(h.exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Connecting to DB")
	db, err := storage.New(h.config.DatabaseFilePath)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	h.storage = db

	slog.Info("Starting schedules")
	if err := h.startSchedules(); err != nil {
		return fmt.Errorf("start schedules: %w", err)
	}

	h.runHTTP()

	slog.Info(fmt.Sprintf("Server started after %s", time.Since(startupStart)))

	close(h.started)

	h.resumePendingTestRuns()

	signal := <-h.exit

	h.gracefulShutdown(signal)

	return nil
}

// ServerPort returns the port that the server is using. This is useful
// when the port is randomly allocated on startup.
func (h *Server) ServerPort() int {
	return h.config.Port
}

// WaitForStartup blocks until the server has started up.
func (h *Server) WaitForStartup() {
	<-h.started
}

// Shutdown shuts down the server and blocks until it is finished.
func (h *Server) Shutdown() {
	h.exit <- os.Interrupt
	<-h.shutdown
}

func (h *Server) loadLibraryFiles() error {
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

		h._userProvidedTestSuites = append(h._userProvidedTestSuites, suites...)
		h.schedules = append(h.schedules, schedules...)
	}

	return nil
}

func (h *Server) resumePendingTestRuns() {
	ctx := context.Background()

	pendingRuns, err := h.storage.LoadPendingTestSuiteRuns(ctx)
	if err != nil {
		slog.Warn("Unable to load pending test suite runs", "error", err)
		return
	}

	for _, tsr := range pendingRuns {
		testSuite, ok := h.readOnlyTestSuites[tsr.SuiteName]
		if !ok {
			slog.Warn("Cannot continue pending test suite run, missing test suite", "suite-name", tsr.SuiteName)
			continue
		}

		tsr.TestResults, err = h.storage.LoadTestRuns(ctx, tsr.SuiteName, tsr.ID)
		if err != nil {
			slog.Warn("Unable to load pending test suite test runs", "error", err)
			continue
		}

		go h.runTestSuite(testSuite, tsr, nil, false)
	}

	if len(pendingRuns) > 0 {
		slog.Info(fmt.Sprintf("Resumed %d test suite run(s) with pending tests", len(pendingRuns)))
	}
}

func (s *Server) gracefulShutdown(signal os.Signal) {
	slog.Info("Received signal, shutting down", "signal", signal.String())

	s.shuttingDown = true

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

	close(s.shutdown)

	slog.Info("DB connection closed")
	slog.Info("Shutdown successful")
}

func (s *Server) printTestSuites() {
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

func (s *Server) startSchedules() error {
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
func (s *Server) startNewTestSuiteRun(ts model.TestSuite, triggeredBy string, testFilter *regexp.Regexp) (model.TestSuiteRun, error) {
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
func (s *Server) runTestSuite(
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

		if s.shuttingDown {
			// we will continue pending test runs on restart
			continue
		}
		s.runTest(suite, &tsr, tr)
	}

	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			slog.Warn("teardown of suite failed", "suite-name", suite.Name, "error", err)
		}
	}

	// TODO: Add the plugin context to the `testRunFinishedEvent`
	s.plugins.notifyTestSuiteFinished(suite, tsr)

	tsr.Result = tsr.ResultFromTestResults()
	if tsr.Result != model.ResultPending {
		tsr.End = time.Now()
	}
	tsr.DurationInMS = tsr.TestSuiteDuration()

	if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
		slog.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
	}

	if tsr.Result != model.ResultPending {
		s.plugins.notifyTestSuiteFinishedAsync(suite, tsr)

		metric.TestSuitesRun.WithLabelValues(suite.Namespace, suite.Name, string(tsr.Result)).Inc()
	}
}

// runTest runs an individual test that is part of a test suite. This function must only be called
// by `runTestSuite()`.
func (s *Server) runTest(suite model.TestSuite, testSuiteRun *model.TestSuiteRun, testRun *model.TestRun) {
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
func (s *Server) asyncPluginCallback(p Plugin, pluginContext map[string]any) {
	// todo
}

func (h *Server) mapTestSuites() error {
	for _, ts := range h._userProvidedTestSuites {
		if ts.Name == "" {
			return errors.New("test suite name is not set")
		}
		if len(ts.Tests) == 0 {
			return fmt.Errorf("test suite %s has no tests configured", ts.Name)
		}
		if _, ok := h.readOnlyTestSuites[ts.Name]; ok {
			return fmt.Errorf("duplicate test suite with name %s", ts.Name)
		}

		mappedTs := model.TestSuite{
			Name:      ts.Name,
			Namespace: ts.Namespace,
			Setup:     ts.Setup,
			Teardown:  ts.Teardown,
			Tests:     make(map[string]model.TestFunc),
		}

		for _, t := range ts.Tests {
			mappedTs.Tests[testName(t)] = t
		}

		h.readOnlyTestSuites[mappedTs.Name] = mappedTs
	}

	return nil
}

func testName(tf TestFunc) string {
	fullFuncName := runtime.FuncForPC(reflect.ValueOf(tf).Pointer()).Name()

	packageIndex := strings.LastIndex(fullFuncName, ".") + 1
	// remove the package name
	return fullFuncName[packageIndex:]
}
