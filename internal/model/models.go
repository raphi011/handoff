// The `model`s package is very atypical for projects written in go, but unfortunately
// cannot be avoided as it helps to avoid cyclic dependencies. Types required by a library user
// such as `TestFunc` are reexported by the handoff package.
package model

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
)

type TestSuiteRun struct {
	// ID is the identifier of the test run.
	ID int `json:"id"`
	// SuiteName is the name of the test suite that is run.
	SuiteName string `json:"suiteName"`
	// Result is the outcome of the entire test suite run.
	Result Result `json:"result"`
	// Tests counts the total amount of tests in the suite.
	Tests int `json:"tests"`
	// Flaky is set to true if one or more tests only succeed
	// after being retried.
	Flaky bool `json:"flaky"`
	// Params passed in to a run.
	Params RunParams `json:"params"`
	// Scheduled is the time when the test was triggered, e.g.
	// through a http call.
	Scheduled time.Time `json:"scheduled"`
	// Start is the time when the test run first started executing.
	Start time.Time `json:"start"`
	// End is the time when the test run finished executing.
	End time.Time `json:"end"`
	// DurationInMS is the test run execution times summed up.
	DurationInMS int64 `json:"durationInMs"`
	// SetupLogs are the logs that are written during the setup phase.
	SetupLogs string `json:"setupLogs"`
	// Environment is additional information on where the tests are run (e.g. cluster name).
	Environment string `json:"environment"`
	// TestResults contains the detailed test results of each test.
	TestResults []TestRun `json:"testResults"`
}

type ScheduledRun struct {
	// Name is the name of the schedule.
	Name string
	// TestSuiteName is the name of the test suite to be run.
	TestSuiteName string
	// Schedule defines how often a run is scheduled. For the format see
	// https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format
	Schedule string
	// TestFilter allows enabling/filtering only certain tests of a testsuite to be run
	TestFilter *regexp.Regexp

	// EntryID identifies the cronjob
	EntryID cron.EntryID
}

type RunParams struct {
	// TriggeredBy denotes the origin of the test run, e.g. scheduled or via http call.
	TriggeredBy string
	// Reference can be set when starting a new test suite run to identify
	// a test run by a user provided value.
	Reference       string
	MaxTestAttempts int
	Timeout         time.Duration
	// TestFilter filters out a subset of the tests and skips the remaining ones.
	TestFilter *regexp.Regexp
}

func (tsr TestSuiteRun) Copy() TestSuiteRun {
	tsrCopy := tsr
	tsrCopy.TestResults = []TestRun{}

	for _, tr := range tsr.TestResults {
		tsrCopy.TestResults = append(tsrCopy.TestResults, tr.Copy())
	}

	return tsrCopy

}

func (t TestSuiteRun) ShouldRetry(tr TestRun) bool {
	return tr.Result == ResultFailed && tr.Attempt < t.Params.MaxTestAttempts
}

func (tsr TestSuiteRun) ResultFromTestResults() Result {
	result := ResultPassed

	for _, r := range tsr.latestTestAttempts() {
		if r.Result == ResultPending {
			return ResultPending
		}

		if r.Result == ResultFailed && !r.SoftFailure {
			result = ResultFailed
		}
	}

	return result
}

func (tsr TestSuiteRun) LatestTestAttempts() []TestRun {
	var runs []TestRun

	for _, tr := range tsr.latestTestAttempts() {
		runs = append(runs, tr)
	}

	sort.Slice(runs, func(x, y int) bool {
		return runs[x].Name < runs[y].Name
	})

	return runs
}

func (tsr TestSuiteRun) latestTestAttempts() map[string]TestRun {
	latestAttempts := map[string]TestRun{}

	for i := range tsr.TestResults {
		tr := tsr.TestResults[i]

		if trMap, ok := latestAttempts[tr.Name]; !ok || trMap.Attempt < tr.Attempt {
			latestAttempts[tr.Name] = tr
		}
	}

	return latestAttempts
}

func (tsr TestSuiteRun) TestRunsByName(testName string) []TestRun {
	runs := []TestRun{}

	for _, tr := range tsr.TestResults {
		if tr.Name == testName {
			runs = append(runs, tr)
		}
	}

	return runs
}

func (tsr TestSuiteRun) IsFlaky() bool {
	for _, r := range tsr.TestResults {
		// todo: make sure >= 2 attempts have failed
		// (one could have been skipped as well)
		if r.Attempt > 1 {
			return true
		}
	}

	return false
}

func (tsr TestSuiteRun) PendingTests() []*TestRun {
	pendingRuns := []*TestRun{}

	for i := range tsr.TestResults {
		if tsr.TestResults[i].Result == ResultPending {
			pendingRuns = append(pendingRuns, &tsr.TestResults[i])
		}
	}

	return pendingRuns
}

func (tsr TestSuiteRun) TestSuiteDuration() int64 {
	duration := int64(0)

	for _, r := range tsr.latestTestAttempts() {
		duration += r.DurationInMS
	}

	return duration
}

type TestRun struct {
	SuiteName  string `json:"suiteName"`
	SuiteRunID int    `json:"suiteRunId"`
	Name       string `json:"name"`
	// Result is the outcome of the test run.
	Result Result `json:"result"`
	// Test run attempt counter
	Attempt int `json:"attempt"`
	// SoftFailure if set to true, does not fail a test suite when the test run fails.
	SoftFailure bool `json:"softFailure"`
	// Logs contains log messages written by the test itself.
	Logs string `json:"logs"`
	// Start marks the start time of the test run.
	Start time.Time `json:"start"`
	// End marks the end time of the test run.
	End time.Time `json:"end"`
	// DurationInMS is the duration of the test run in milliseconds (end-start).
	DurationInMS int64 `json:"durationInMs"`
	// Spans are timings that can be created within a test that give more information
	// on how long certain parts of a test took.
	Spans []*Span `json:"spans"`
	// Context contains additional testrun specific information that is collected during and
	// after a test run either by the test itself (`t.SetContext`) or via plugins. This can
	// e.g. contain correlation ids or links to external services that may help debugging a test run
	// (among other things).
	Context TestContext `json:"context"`
}

type Span struct {
	Start   time.Time   `json:"start"`
	End     *time.Time  `json:"end"`
	Name    string      `json:"name"`
	Context TestContext `json:"context"`
}

func (s *Span) String() string {
	t := s.End.Sub(s.Start).Milliseconds()
	return fmt.Sprintf("%s (%d ms) %+v", s.Name, t, s.Context)
}

func StartSpan(name string, kv ...any) (*Span, error) {
	context := make(map[string]any)

	if len(kv)%2 != 0 {
		return nil, errors.New("Uneven number of key-value context arguments")
	}

	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			return nil, fmt.Errorf("Key %v is not a string", key)
		}

		context[key] = kv[i+1]
	}

	return &Span{
		Start:   time.Now(),
		Name:    name,
		Context: context,
	}, nil
}

func (s *Span) EndSpan() {
	if s.End == nil {
		now := time.Now()
		s.End = &now
	}
}

func (s *Span) AddContext(key string, value any) {
	s.Context[key] = value
}

func (tr TestRun) Copy() TestRun {
	trCopy := tr
	// todo: copy values? We currently only need this during
	// test suite run where the context is empty, this might change though.
	trCopy.Context = make(TestContext)
	return trCopy
}

func (t TestRun) NewAttempt() TestRun {
	return TestRun{
		SuiteName:  t.SuiteName,
		SuiteRunID: t.SuiteRunID,
		Name:       t.Name,
		Result:     ResultPending,
		Attempt:    t.Attempt + 1,
	}
}

type Result string

const (
	ResultPending Result = "pending"
	ResultSkipped Result = "skipped"
	ResultPassed  Result = "passed"
	ResultFailed  Result = "failed"
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
	// TestRetries is the amount of times a test is retried when failing.
	MaxTestAttempts int
	// Namespace allows grouping of test suites, e.g. by team name.
	Namespace string
	Setup     func() error
	Teardown  func() error
	Tests     map[string]TestFunc
	// lock      *sync.Mutex
}

func (t TestSuite) SafeTeardown() (err error) {
	if t.Teardown == nil {
		return nil
	}

	defer func() {
		r := recover()

		if r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	err = t.Teardown()
	return
}

func (t TestSuite) SafeSetup() (err error) {
	if t.Setup == nil {
		return nil
	}

	defer func() {
		r := recover()

		if r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	// TODO: allow setup to add values to the runtime context?
	err = t.Setup()
	return
}

func (t TestSuite) FilterTests(filter *regexp.Regexp) []string {
	tests := []string{}

	for testName := range t.Tests {
		if filter != nil && !filter.MatchString(testName) {
			continue
		}
		tests = append(tests, testName)
	}

	return tests
}

// TB is a carbon copy of the stdlib testing.TB interface + some custom handoff functions. Unfortunately we cannot reuse
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

	/* Handoff specific */
	SoftFailure()
	Attempt() int
	StartSpan(name string, kv ...any) *Span
}
