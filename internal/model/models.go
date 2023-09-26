// The `model`s package is very atypical for projects written in go, but unfortunately
// cannot be avoided as it helps to avoid cyclic dependencies. Types required by a library user
// such as `TestFunc` are reexported by the handoff package.
package model

import (
	"regexp"
	"time"
)

type TestSuiteRun struct {
	// ID is the identifier of the test run.
	ID int `json:"id"`
	// SuiteName is the name of the test suite that is run.
	SuiteName string `json:"suiteName"`
	// TestResults contains the detailed test results of each test.
	TestResults []TestRun `json:"testResults"`
	// Result is the outcome of the entire test run.
	Result Result `json:"result"`
	// TestFilter filters out a subset of the tests and skips the
	// remaining ones (not implemented yet).
	TestFilter      string `json:"testFilter"`
	TestFilterRegex *regexp.Regexp
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

type TestRun struct {
	Name string `json:"name"`
	// Passed denotes if the test has passed.
	Passed bool `json:"passed"`
	// Skipped denotes if the test was skipped (e.g. by calling t.Skip()).
	Skipped bool `json:"skipped"`
	// Logs contains log messages written by the test itself.
	Logs []string `json:"logs"`
	// Start marks the start time of the test run.
	Start time.Time `json:"start"`
	// End marks the end time of the test run.
	End time.Time `json:"end"`
	// DurationInMS is the duration of the test run in milliseconds (end-start).
	DurationInMS int64 `json:"durationInMs"`
	// RunContext is data that can be set by the test. This can be used
	// to add additional context to the test run, e.g. correlation ids.
	RunContext map[string]any `json:"runContext"`
	// PluginContext contains per plugin information that was created from
	// a test run and can be used to show additional information to a developer.
	PluginContext map[string]any
}

type Result string

const (
	ResultSkipped     Result = "skipped"
	ResultPassed      Result = "passed"
	ResultFailed      Result = "failed"
	ResultSetupFailed Result = "setup-failed"
)

type TestFunc func(t TB)

// TestSuite is a static definition of a testsuite and contains
// Setup, Teardown and a collection of Test functions.
type TestSuite struct {
	// Name of the testsuite
	Name string `json:"name"`
	// AssociatedService can optionally contain the name of the service
	// that this testsuite is associated with. This will be used for metric labels.
	AssociatedService string `json:"associatedService"`
	Setup             func() error
	Teardown          func() error
	Tests             []Test
}

type Test struct {
	// The name of the function which is set by fetching the functions name via reflection.
	Name string
	Func TestFunc
}

// TB is a carbon copy of the stdlib testing.TB interface. Unfortunately we cannot reuse
// the original testing.TB interface because it deliberately includes the `private()` function
// to prevent others from implementing it to allow them to add new functions over time without
// breaking anything.
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
