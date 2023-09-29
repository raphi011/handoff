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
	// Result is the outcome of the entire test suite run.
	Result Result `json:"result"`
	// TestFilter filters out a subset of the tests and skips the
	// remaining ones (not implemented yet).
	TestFilter      string `json:"testFilter"`
	TestFilterRegex *regexp.Regexp
	// Tests counts the total amount of tests in the suite.
	Tests int `json:"tests"`
	// Scheduled is the time when the test was triggered, e.g.
	// through a http call.
	Scheduled time.Time `json:"scheduled"`
	// Start is the time when the test run started executing.
	Start time.Time `json:"start"`
	// End is the time when the test run finished executing.
	End time.Time `json:"end"`
	// DurationInMS is the test run execution times summed up.
	DurationInMS int64 `json:"durationInMs"`
	// SetupLogs are the logs that are written during the setup phase.
	SetupLogs string `json:"setupLogs"`
	// TriggeredBy denotes the origin of the test run, e.g. scheduled or via http call.
	TriggeredBy string `json:"triggeredBy"`
	// Environment is additional information on where the tests are run (e.g. cluster name).
	Environment string `json:"environment"`
	// TestResults contains the detailed test results of each test.
	TestResults []TestRun `json:"testResults"`
}

type TestRun struct {
	SuiteName  string `json:"suiteName"`
	SuiteRunID int    `json:"suiteRunId"`
	Name       string `json:"name"`
	// Result is the outcome of the test run.
	Result Result `json:"result"`
	// Test run attempt counter
	Attempt int `json:"attempt"`
	// Was this test attempt manually rerun/forced?
	Forced bool `json:"forced"`
	// Logs contains log messages written by the test itself.
	Logs string `json:"logs"`
	// Start marks the start time of the test run.
	Start time.Time `json:"start"`
	// End marks the end time of the test run.
	End time.Time `json:"end"`
	// DurationInMS is the duration of the test run in milliseconds (end-start).
	DurationInMS int64 `json:"durationInMs"`
	// Context contains additional testrun specific information that is collected during and
	// after a test run either by the test itself (`t.SetContext`) or via plugins. This can
	// e.g. contain correlation ids or links to external services that may help debugging a test run
	// (among other things).
	Context TestContext `json:"context"`
}

func (t TestRun) NewForcedAttempt() TestRun {
	return TestRun{
		SuiteName:  t.SuiteName,
		SuiteRunID: t.SuiteRunID,
		Name:       t.Name,
		Result:     ResultPending,
		Forced:     true,
		Start:      time.Now(),
	}
}

type Result string

const (
	ResultPending     Result = "pending"
	ResultSkipped     Result = "skipped"
	ResultPassed      Result = "passed"
	ResultFailed      Result = "failed"
	ResultSetupFailed Result = "setup-failed"
)

type TestContext map[string]any

func (c TestContext) Merge(c2 TestContext) {
	// TODO merge c2 into c1
}

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
	Tests             map[string]TestFunc
}

func (t TestSuite) FilterTests(filter *regexp.Regexp) map[string]TestFunc {
	if filter == nil {
		return t.Tests
	}

	filteredtests := map[string]TestFunc{}

	for testName, testFunc := range t.Tests {
		if filter.MatchString(testName) {
			filteredtests[testName] = testFunc
		}
	}

	return filteredtests
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
