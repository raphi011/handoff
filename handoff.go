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
	"sync/atomic"
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

	shuttingDown bool

	// started will be closed when the service has started.
	started chan any
	// shutdown will be closed when the service has shut down.
	shutdown chan error

	// exit receives os signals
	exit chan os.Signal

	// cron object used for scheduled runs
	cron *cron.Cron

	// a temp cache of currently executing test suite runs
	tempTestSuiteRuns *storage.TsrCache

	runningTestSuites sync.WaitGroup

	httpServer *http.Server

	suiteID atomic.Uint64

	log *slog.Logger

	storage storage.Storage
}

type config struct {
	HostIP string `arg:"-h,--host,env:HANDOFF_HOST" help:"ip address the server should bind to" default:"localhost"`

	// Port for the web api
	Port int `arg:"-p,env:HANDOFF_PORT" help:"port used by the server (server mode only)" default:"1337"`

	EnablePprof bool `arg:"--enable-pprof" help:"enable pprof debugging endpoints" default:"false"`

	EnableBadgerDb bool `arg:"--enable-badger" help:"enable experimental badger db" default:"false"`

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
		tempTestSuiteRuns:       storage.NewTsrCache(),
		started:                 make(chan any),
		exit:                    make(chan os.Signal, 1),
		shutdown:                make(chan error, 1),
		log:                     slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	s.hooks = newHookManager(s.asyncHookCallback)

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Server) Run() error {
	startupStart := time.Now()

	// we want to make sure that the test suite functions only
	// log using the functions provided through the t struct
	// and not 'pollute' the server logs, so we need to redirect
	// the standard test loggers to /dev/null and use a custom one
	// for the server.
	stdliblog.SetOutput(io.Discard)
	defer stdliblog.SetOutput(os.Stderr)

	arg.MustParse(&s.config)

	if err := s.mapTestSuites(); err != nil {
		return err
	}

	if s.config.ListTestSuites {
		s.printTestSuites()
	}

	if err := s.hooks.init(); err != nil {
		return fmt.Errorf("init hooks: %w", err)
	}

	signal.Notify(s.exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var err error

	if s.config.EnableBadgerDb {
		s.storage, err = storage.NewBadgerStorage(s.config.DatabaseFilePath, s.log)
		if err != nil {
			return fmt.Errorf("init badger storage: %w", err)
		}
	} else {
		s.storage, err = storage.NewSqlite(s.config.DatabaseFilePath, s.log)
		if err != nil {
			return fmt.Errorf("init sqlte storage: %w", err)
		}
	}

	if err := s.startSchedules(); err != nil {
		return fmt.Errorf("start schedules: %w", err)
	}

	if err = s.runHTTP(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}

	s.log.Info(fmt.Sprintf("Server started after %s", time.Since(startupStart)))

	close(s.started)

	s.resumePendingTestRuns()

	signal := <-s.exit

	s.gracefulShutdown(signal)

	return nil
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
	s.exit <- os.Interrupt
	return <-s.shutdown
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

		tsr.TestResults, err = s.storage.LoadTestRuns(ctx, tsr.SuiteName, tsr.ID)
		if err != nil {
			s.log.Warn("Unable to load pending test suite test runs", "error", err)
			continue
		}

		continued++

		go s.runTestSuite(testSuite, tsr)
	}

	if continued > 0 {
		s.log.Info(fmt.Sprintf("Resumed %d test suite run(s) with pending tests", continued))
	}
}

func (s *Server) gracefulShutdown(signal os.Signal) {
	s.log.Info("Received signal, shutting down", "signal", signal.String())

	s.shuttingDown = true

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

	s.shutdown <- err
	close(s.shutdown)
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

	if option.MaxTestAttempts == 0 {
		option.MaxTestAttempts = ts.MaxTestAttempts
	}

	suiteID := s.suiteID.Add(1)

	tsr := model.TestSuiteRun{
		// TODO lock test suite and fetch latest id from storage.
		ID:          int(suiteID),
		SuiteName:   ts.Name,
		TestResults: []*model.TestRun{},
		Result:      model.ResultPending,
		Params:      option,
		Tests:       len(ts.Tests),
		Scheduled:   time.Now(),
		Environment: s.config.Environment,
	}

	ctx, err := s.storage.StartTransaction(context.Background())
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to start transaction: %w", err)
	}
	defer s.storage.RollbackTransaction(ctx)

	err = s.storage.InsertTestSuiteRun(ctx, tsr)
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to persist new test suite run: %w", err)
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

		err = s.storage.InsertTestRun(ctx, tr)
		if err != nil {
			return model.TestSuiteRun{}, fmt.Errorf("unable to persist new test run: %w", err)
		}

		tsr.TestResults = append(tsr.TestResults, &tr)
	}

	if err = s.storage.CommitTransaction(ctx); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("unable to persist the test suite run: %w", err)
	}

	go s.runTestSuite(ts, tsr)

	return tsr, nil
}

// runTestSuite executes a test suite run. It will run all tests that are either pending
// or can be retried (attempt<maxattempts).
func (s *Server) runTestSuite(
	suite model.TestSuite,
	tsr model.TestSuiteRun,
) {
	if tsr.Result == model.ResultPassed {
		// nothing to do here
		return
	}

	log := s.log.With("suite-name", suite.Name, "run-id", tsr.ID)

	ctx := context.Background()

	s.runningTestSuites.Add(1)
	defer s.runningTestSuites.Done()

	timeNotSet := time.Time{}
	if tsr.Start == timeNotSet {
		tsr.Start = time.Now()
		if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
			log.Error("updating test suite run failed", "error", err)
		}
	}

	testSuitesRunning := metric.TestSuitesRunning.WithLabelValues(suite.Namespace, suite.Name)
	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if err := suite.SafeSetup(); err != nil {
		log.Warn("setup of suite failed", "error", err)
		end := time.Now()

		tsr.Result = model.ResultFailed
		tsr.End = end
		tsr.SetupLogs = fmt.Sprintf("setup failed: %v", err)

		for _, tr := range tsr.LatestTestAttempts() {
			if tr.Result == model.ResultPending {
				tr.Result = model.ResultSkipped
				tr.Logs = "test suite run setup failed: skipped"
				tr.End = end

				if err = s.storage.UpdateTestRun(ctx, *tr); err != nil {
					log.Error("could not mark test run as skipped after test suite setup failed", "error", err)
				}
			}
		}

		if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
			log.Error("updating test suite run failed", "error", err)
		}

		return
	}

	testsToRun := tsr.PendingTests()

	for i := 0; i < len(testsToRun); i++ {
		tr := testsToRun[i]

		if s.shuttingDown {
			// we will continue pending test runs on restart
			continue
		}

		s.runTest(ctx, suite, &tsr, tr)

		if tsr.ShouldRetry(*tr) {
			newAttempt := tr.NewAttempt()

			if err := s.storage.InsertTestRun(ctx, newAttempt); err != nil {
				log.Error("could not insert new test run attempt", "error", err)
				return
			}
			testsToRun = append(testsToRun, &newAttempt)
			tsr.TestResults = append(tsr.TestResults, &newAttempt)
		}
	}

	if err := suite.SafeTeardown(); err != nil {
		log.Warn("teardown of suite failed", "error", err)
	}

	// TODO: Add the plugin context to the `testRunFinishedEvent`
	s.hooks.notifyTestSuiteFinished(suite, tsr)

	tsr.Result = tsr.ResultFromTestResults()
	if tsr.Result != model.ResultPending {
		tsr.End = time.Now()
	}
	tsr.DurationInMS = tsr.TestSuiteDuration()
	tsr.Flaky = tsr.IsFlaky()

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
func (s *Server) runTest(ctx context.Context, suite model.TestSuite, testSuiteRun *model.TestSuiteRun, testRun *model.TestRun) {
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

		s.hooks.notifyTestFinished(suite, *testSuiteRun, testRun.Name, t.runtimeContext)

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

		if err := s.storage.UpdateTestRun(ctx, *testRun); err != nil {
			s.log.Error("updating test suite run failed", "suite-name", suite.Name, "error", err)
		}

		s.hooks.notifyTestFinishedAync(suite, *testSuiteRun, testRun.Name, t.runtimeContext)

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
