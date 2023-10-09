package handoff

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdliblog "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
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

	// configured hooks
	hooks *hookManager

	// a map of all testsuites that must not be modified after
	// initialisation.
	readOnlyTestSuites map[string]model.TestSuite

	// _userProvidedTestSuites is a list of all test suites provided
	// by the user and will be mapped to `readOnlyTestSuites` on startup.
	_userProvidedTestSuites []TestSuite

	// scheduled runs configured by the user
	schedules []ScheduledRun

	// started will be closed when the service has started.
	started chan any

	// hasShutdown will be closed when the service has shut down.
	hasShutdown chan error

	// shutdown signals the server to shut down.
	shutdown chan any

	// cron object used for scheduled runs
	cron *cron.Cron

	// a temp cache of currently executing test suite runs
	// tempTestSuiteRuns *storage.TsrCache

	runningTestSuites sync.WaitGroup

	httpServer *http.Server

	log *slog.Logger

	storage *storage.BadgerStorage
}

type config struct {
	HostIP string `arg:"-h,--host,env:HANDOFF_HOST" help:"ip address the server should bind to" default:"localhost"`

	// Port for the web api
	Port int `arg:"-p,env:HANDOFF_PORT" help:"port used by the server (server mode only)" default:"1337"`

	EnableDebugLogging bool `arg:"--debug-log" help:"enable debug level logging" default:"false"`

	EnablePprof bool `arg:"--enable-pprof" help:"enable pprof debugging endpoints" default:"false"`

	// location of the sqlite database file, if empty we default
	// to an in-memory database.
	DatabaseFilePath string `arg:"-d,--database,env:HANDOFF_DATABASE" help:"database file location" default:"handoff.db"`

	// List will, if set to true, print all loaded test suites
	// and immediately exit.
	ListTestSuites bool `arg:"-l,--list" help:"list all configured test suites, schedules and hooks and exit" default:"false"`

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
	Namespace       string
	MaxTestAttempts int
	Setup           func() error
	Teardown        func() error
	Timeout         time.Duration
	Tests           []TestFunc
}

// Reexport to allow library users to reference these types
type TestFunc = model.TestFunc
type TB = model.TB
type TestContext = model.TestContext

type Option func(s *Server)

var (
	registeredSuites    []TestSuite
	registeredSchedules []ScheduledRun
)

// Register registers test suites and schedules to be loaded when `*server.New()` is called.
func Register(suites []TestSuite, schedules []ScheduledRun) {
	registeredSuites = append(registeredSuites, suites...)
	registeredSchedules = append(registeredSchedules, schedules...)
}

// New configures a new Handoff instance.
func New(opts ...Option) *Server {
	s := &Server{
		_userProvidedTestSuites: registeredSuites,
		schedules:               registeredSchedules,
		readOnlyTestSuites:      map[string]model.TestSuite{},
		// tempTestSuiteRuns:       storage.NewTsrCache(),
		started:     make(chan any),
		hasShutdown: make(chan error, 1),
		shutdown:    make(chan any),
		log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	s.hooks = newHookManager(s.asyncHookCallback)

	for _, o := range opts {
		o(s)
	}

	return s
}

// Run runs the server with the passed in flags. Usually you want to pass in
// `os.Args` here.
func (s *Server) Run(args []string) error {
	startupStart := time.Now()

	// we want to make sure that the test suite functions only
	// log using the functions provided through the t struct
	// and not 'pollute' the server logs, so we need to redirect
	// the standard test loggers to /dev/null and use a custom one
	// for the server.
	stdliblog.SetOutput(io.Discard)
	defer stdliblog.SetOutput(os.Stderr)

	s.parseConfig(args)

	if err := s.mapTestSuites(); err != nil {
		return err
	}

	if s.config.ListTestSuites {
		s.printTestSuites()
	}

	if err := s.hooks.init(); err != nil {
		return fmt.Errorf("init hooks: %w", err)
	}

	s.signalHandler()

	storage, err := storage.NewBadgerStorage(s.config.DatabaseFilePath, s.log)
	if err != nil {
		return fmt.Errorf("init badger storage: %w", err)
	}
	s.storage = storage

	if err := s.startSchedules(); err != nil {
		return fmt.Errorf("start schedules: %w", err)
	}

	if err = s.runHTTP(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}

	s.log.Info(fmt.Sprintf("Server started after %s", time.Since(startupStart)))

	close(s.started)

	s.resumePendingTestRuns()

	<-s.shutdown

	s.gracefulShutdown()

	return nil
}

// parseConfig was more or less copied from arg.MustParse()
func (s *Server) parseConfig(args []string) {
	program := "handoff"
	if len(args) > 0 {
		program = args[0]
		args = args[1:]
	}

	p, err := arg.NewParser(arg.Config{Program: program}, &s.config)
	if err != nil {
		s.log.Error("Unable to create args parser", "error", err)
		os.Exit(-1)
	}

	err = p.Parse(args)
	switch {
	case err == arg.ErrHelp:
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	case err == arg.ErrVersion:
		fmt.Fprintln(os.Stdout, s.config.Version())
		os.Exit(0)
	case err != nil:
		p.Fail(err.Error())
	}
}

// ServerPort returns the port that the server is using. This is useful
// when the port is randomly allocated on startup.
func (h *Server) ServerPort() int {
	return h.config.Port
}

// WaitForStartup blocks until the server has started up.
func (s *Server) WaitForStartup() {
	// TODO: maybe respond with the error if startup fails?
	<-s.started
}

// Shutdown shuts down the server and blocks until it is finished.
func (s *Server) Shutdown() error {
	close(s.shutdown)
	return <-s.hasShutdown
}

func (s *Server) signalHandler() {
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		r := <-signalChan
		s.log.Info("Received signal, shutting down", "signal", r.String())

		close(s.shutdown)
	}()
}

func (s *Server) resumePendingTestRuns() {
	ctx := context.Background()

	pendingRuns, err := s.storage.LoadPendingTestSuiteRuns(ctx)
	if err != nil {
		s.log.Warn("Unable to load pending test suite runs", "error", err)
		return
	}

	continued := 0

	for _, tsr := range pendingRuns {
		testSuite, ok := s.readOnlyTestSuites[tsr.SuiteName]
		if !ok {
			// TODO: shall we mark these as failed as we cannot continue this run?
			s.log.Warn("Cannot continue pending test suite run, missing test suite", "suite-name", tsr.SuiteName)
			continue
		}

		continued++

		go s.runTestSuite(testSuite, tsr)
	}

	if continued > 0 {
		s.log.Info(fmt.Sprintf("Resumed %d test suite run(s) with pending tests", continued))
	}
}

func (s *Server) gracefulShutdown() {
	httpStopped := s.stopHTTP()
	cronStopCtx := s.cron.Stop()

	httpErr := <-httpStopped
	s.log.Info("Http listener stopped")
	<-cronStopCtx.Done()
	s.log.Info("Scheduled tests stopped")

	pluginStopCtx := s.hooks.shutdown()
	<-pluginStopCtx.Done()
	s.log.Info("Plugins stopped")

	s.runningTestSuites.Wait()
	s.log.Info("Running test suites finished")

	dbErr := s.storage.Close()
	s.log.Info("DB closed")

	err := errors.Join(httpErr, dbErr)

	s.hasShutdown <- err
	close(s.hasShutdown)
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

		ts, ok := s.readOnlyTestSuites[schedule.TestSuiteName]
		if !ok {
			return fmt.Errorf("test suite %q not found", schedule.TestSuiteName)
		}

		if len(ts.FilterTests(schedule.TestFilter)) == 0 {
			return errors.New("no tests match filter regex %s")
		}

		entryID, err := s.cron.AddFunc(schedule.Schedule, func() {
			_, err := s.startNewTestSuiteRun(ts, model.RunParams{
				TriggeredBy:     "scheduled",
				TestFilter:      schedule.TestFilter,
				MaxTestAttempts: ts.MaxTestAttempts,
			})
			if err != nil {
				s.log.Error("starting new scheduled test suite run failed", "error", err, "test-suite", ts.Name)
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
func (s *Server) startNewTestSuiteRun(ts model.TestSuite, option model.RunParams) (model.TestSuiteRun, error) {
	// if option.Reference == "" {
	// TODO: should we generate a new random reference when none is passed?
	// OR e.g. base64 encode suitename+initial-run-id
	// }
	if s.isShuttingDown() {
		return model.TestSuiteRun{}, errors.New("shutting down")
	}

	if option.MaxTestAttempts == 0 {
		option.MaxTestAttempts = ts.MaxTestAttempts
	}

	tsr := model.TestSuiteRun{
		// TODO lock test suite and fetch latest id from storage.
		SuiteName:   ts.Name,
		TestResults: []model.TestRun{},
		Result:      model.ResultPending,
		Params:      option,
		Tests:       len(ts.Tests),
		Scheduled:   time.Now(),
		Environment: s.config.Environment,
	}

	for testName := range ts.Tests {
		result := model.ResultPending

		if tsr.Params.TestFilter != nil && !tsr.Params.TestFilter.MatchString(testName) {
			result = model.ResultSkipped
		}

		tr := model.TestRun{
			SuiteName:  tsr.SuiteName,
			SuiteRunID: tsr.ID,
			Name:       testName,
			Result:     result,
			Attempt:    1,
			Context:    model.TestContext{},
		}

		tsr.TestResults = append(tsr.TestResults, tr)
	}

	ctx := context.Background()

	var err error

	tsr.ID, err = s.storage.InsertTestSuiteRun(ctx, tsr)
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to persist new test suite run: %w", err)
	}

	tsrCopy := tsr.Copy()

	go s.runTestSuite(ts, tsr)

	// return a copy otherwise we might get a data race when marshalling the testresults
	// and running the tests
	return tsrCopy, nil
}

func (s *Server) isShuttingDown() bool {
	select {
	case <-s.shutdown:
		return true
	default:
		return false
	}
}

// runTestSuite executes a test suite run. It will run all tests that are either pending
// or can be retried (attempt<maxattempts).
func (s *Server) runTestSuite(
	suite model.TestSuite,
	tsr model.TestSuiteRun,
) {
	s.runningTestSuites.Add(1)
	defer s.runningTestSuites.Done()

	if tsr.Result == model.ResultPassed {
		// nothing to do here
		return
	}

	ctx := context.Background()

	log := s.log.With("suite-name", suite.Name, "run-id", tsr.ID)

	tsr.Start = time.Now()

	testSuitesRunning := metric.TestSuitesRunning.WithLabelValues(suite.Namespace, suite.Name)
	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if err := suite.SafeSetup(); err != nil {
		log.Warn("setup of suite failed", "error", err)
		end := time.Now()

		tsr.Result = model.ResultFailed
		tsr.SetupLogs = fmt.Sprintf("setup failed: %v", err)

		for i := 0; i < len(tsr.TestResults); i++ {
			tr := &tsr.TestResults[i]

			if tr.Result == model.ResultPending {
				tr.Result = model.ResultSkipped
				tr.Logs = "test suite run setup failed: skipped"
				tr.End = end
			}
		}
	}

	// skip if setup failed
	if tsr.Result != model.ResultFailed {
		for i := 0; i < len(tsr.TestResults); i++ {
			tr := &tsr.TestResults[i]

			if s.isShuttingDown() {
				break
			}

			if tr.Result != model.ResultPending {
				continue
			}

			s.runTest(ctx, suite, tsr, tr)

			if tsr.ShouldRetry(*tr) {
				newAttempt := tr.NewAttempt()

				tsr.TestResults = append(tsr.TestResults, newAttempt)
			}
		}

		if err := suite.SafeTeardown(); err != nil {
			log.Warn("teardown of suite failed", "error", err)
		}

		tsr.Result = tsr.ResultFromTestResults()
	}

	if tsr.Result != model.ResultPending {
		tsr.End = time.Now()
	}
	tsr.DurationInMS = tsr.TestSuiteDuration()
	tsr.Flaky = tsr.IsFlaky()

	// TODO: Add the plugin context to the `testRunFinishedEvent`
	s.hooks.notifyTestSuiteFinished(suite, tsr)

	if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
		log.Error("updating test suite run failed", "error", err)
	}

	if tsr.Result != model.ResultPending {
		s.hooks.notifyTestSuiteFinishedAsync(suite, tsr)

		metric.TestSuitesRun.WithLabelValues(suite.Namespace, suite.Name, string(tsr.Result)).Inc()
	}
}

// runTest runs an individual test that is part of a test suite. This function must only be called
// by `runTestSuite()`.
func (s *Server) runTest(
	ctx context.Context,
	suite model.TestSuite,
	testSuiteRun model.TestSuiteRun,
	testRun *model.TestRun,
) {
	t := T{
		attempt:        testRun.Attempt,
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

		s.hooks.notifyTestFinished(suite, testSuiteRun, testRun.Name, t.runtimeContext)

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

		s.hooks.notifyTestFinishedAync(suite, testSuiteRun, testRun.Name, t.runtimeContext)

		if err = t.runTestCleanup(); err != nil {
			s.log.Warn("test cleanup failed", "suite-name", suite.Name, "error", err)
		}
	}()

	suite.Tests[testRun.Name](&t)
}

// asyncHookCallback is called by asynchronous hooks and persists the updated plugincontext change.
func (s *Server) asyncHookCallback(p Hook, pluginContext map[string]any) {
	// todo
}

func (h *Server) mapTestSuites() error {
	if len(h._userProvidedTestSuites) == 0 {
		return errors.New("no test suites provided")
	}

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
			Name:            ts.Name,
			Namespace:       ts.Namespace,
			MaxTestAttempts: ts.MaxTestAttempts,
			Setup:           ts.Setup,
			Teardown:        ts.Teardown,
			Tests:           make(map[string]model.TestFunc),
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

	fqfn := fullFuncName[strings.LastIndex(fullFuncName, "/")+1:]

	parts := strings.Split(fqfn, ".")

	var funcName string

	if len(parts) == 2 {
		// <package-name>.<func-name>
		funcName = parts[1]
	} else {
		// <package-name>.<other-func>.<func-name>.funcX
		funcName = parts[len(parts)-2]
	}

	return funcName
}
