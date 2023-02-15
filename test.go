package handoff

import (
	"fmt"
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
	ID           int32           `json:"id"`
	SuiteName    string          `json:"suiteName"`
	TestResults  []TestRunResult `json:"testResults"`
	Result       Result          `json:"result"`
	TestFilter   string          `json:"testFilter"`
	Tests        int             `json:"tests"`
	Passed       int             `json:"passed"`
	Skipped      int             `json:"skipped"`
	Failed       int             `json:"failed"`
	Scheduled    time.Time       `json:"scheduled"`
	Start        time.Time       `json:"start"`
	End          time.Time       `json:"end"`
	DurationInMS int64           `json:"durationInMs"`
	SetupLogs    []string        `json:"setupLogs"`
}

type TestRunResult struct {
	Name         string    `json:"name"`
	Passed       bool      `json:"passed"`
	Skipped      bool      `json:"skipped"`
	Logs         []string  `json:"logs"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	DurationInMS int64     `json:"durationInMs"`
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
