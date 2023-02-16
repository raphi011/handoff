package handoff

import (
	"fmt"
	"regexp"
	"time"
)

// skipTestErr is passed to panic() to signal
// that a test was skipped.
type skipTestErr struct{}

// failTestErr is passed to panic() to signal
// that a test has failed.
type failTestErr struct{}

type T struct {
	name    string
	logs    []string
	passed  bool
	skipped bool
}

// TB is a carbon copy of the stdlib testing.TB interface
type TB interface {
	Cleanup(func())
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Name() string
	Setenv(key, value string)
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
}

type TestFunc func(t TB)

// TestSuite is a static definition of a testsuite and contains
// Setup, Teardown and a collection of Test functions.
type TestSuite struct {
	Name     string `json:"name"`
	Setup    func() error
	Teardown func() error
	Tests    map[string]TestFunc
}

type Result string

const (
	ResultSkipped     Result = "skipped"
	ResultPassed      Result = "passed"
	ResultFailed      Result = "failed"
	ResultSetupFailed Result = "setup-failed"
)

type TestRun struct {
	// ID is the identifier of the test run.
	ID int32 `json:"id"`
	// SuiteName is the name of the test suite that is run.
	SuiteName string `json:"suiteName"`
	// TestResults contains the detailed test results of each test.
	TestResults []TestRunResult `json:"testResults"`
	// Result is the outcome of the entire test run.
	Result Result `json:"result"`
	// TestFilter filters out a subset of the tests and skips the
	// remaining ones (not implemented yet).
	TestFilter      string `json:"testFilter"`
	testFilterRegex *regexp.Regexp
	// Tests counts the total amount of tests in the suite.
	Tests int `json:"tests"`
	// Passed counts the number of passed tests.
	Passed int `json:"passed"`
	// Skipped counts the number of skipped tests.
	Skipped int `json:"skipped"`
	// Failed counts the number of failed tests.
	Failed int `json:"failed"`
	// Scheduled is the time when the test was triggered, e.g.
	// through a http call.
	Scheduled time.Time `json:"scheduled"`
	// Start is the time when the test run started executing.
	Start time.Time `json:"start"`
	// End is the time when the test run finished executing.
	End time.Time `json:"end"`
	// DurationInMS is the time it took the entire test run to complete (end-start).
	DurationInMS int64 `json:"durationInMs"`
	// SetupLogs are the logs that are written during the setup phase.
	SetupLogs []string `json:"setupLogs"`
	// TriggeredBy denotes the origin of the test run, e.g. scheduled or via http call.
	TriggeredBy string `json:"triggeredBy"`
}

type TestRunResult struct {
	Name    string    `json:"name"`
	Passed  bool      `json:"passed"`
	Skipped bool      `json:"skipped"`
	Logs    []string  `json:"logs"`
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	// CustomData is data that can be set by the test. This can be used
	// to add additional context to the test run, e.g. correlation ids.
	CustomData   map[string]any `json:"customData"`
	DurationInMS int64          `json:"durationInMs"`
}

type Test struct {
	Name  string `json:"name"`
	Suite string
}

func (t *T) Cleanup(func()) {
}

func (t *T) Error(args ...any) {
	t.passed = false
	t.Log(args...)
}

func (t *T) Errorf(format string, args ...any) {
	t.passed = false
	t.Logf(format, args...)
}

func (t *T) Fail() {
	t.passed = false
}

func (t *T) FailNow() {
	t.passed = false
	panic(failTestErr{})
}

func (t *T) Failed() bool {
	return t.passed
}

func (t *T) Fatal(args ...any) {
	t.Error(args...)
	panic(failTestErr{})
}

func (t *T) Fatalf(format string, args ...any) {
	t.Errorf(format, args...)
	panic(failTestErr{})
}

func (t *T) Helper() {}

func (t *T) Log(args ...any) {
	t.logs = append(t.logs, fmt.Sprint(args...))
}

func (t *T) Logf(format string, args ...any) {
	t.logs = append(t.logs, fmt.Sprintf(format, args...))
}

func (t *T) Name() string {
	return t.name
}

func (t *T) Setenv(key, value string) {
}

func (t *T) Skip(args ...any) {
	t.Log(args...)
	t.SkipNow()
}

func (t *T) SkipNow() {
	t.skipped = true
	panic(skipTestErr{})
}

func (t *T) Skipf(format string, args ...any) {
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *T) Skipped() bool {
	return t.skipped
}

func (t *T) TempDir() string {
	return ""
}
