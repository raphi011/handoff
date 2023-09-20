package handoff

import (
	"fmt"

	"github.com/raphi011/handoff/internal/model"
)

// make sure we adhere to the TB interface
var _ TB = &t{}

type t struct {
	name       string
	logs       []string
	passed     bool
	skipped    bool
	runContext map[string]any
}

func (t *t) Cleanup(func()) {
}

func (t *t) Error(args ...any) {
	t.passed = false
	t.Log(args...)
}

func (t *t) Errorf(format string, args ...any) {
	t.passed = false
	t.Logf(format, args...)
}

func (t *t) Fail() {
	t.passed = false
}

func (t *t) FailNow() {
	t.passed = false
	panic(failTestErr{})
}

func (t *t) Failed() bool {
	return t.passed
}

func (t *t) Fatal(args ...any) {
	t.Error(args...)
	panic(failTestErr{})
}

func (t *t) Fatalf(format string, args ...any) {
	t.Errorf(format, args...)
	panic(failTestErr{})
}

func (t *t) Helper() {}

func (t *t) Log(args ...any) {
	t.logs = append(t.logs, fmt.Sprint(args...))
}

func (t *t) Logf(format string, args ...any) {
	t.logs = append(t.logs, fmt.Sprintf(format, args...))
}

func (t *t) Name() string {
	return t.name
}

func (t *t) Setenv(key, value string) {
}

func (t *t) Skip(args ...any) {
	t.Log(args...)
	t.SkipNow()
}

func (t *t) SkipNow() {
	t.skipped = true
	panic(skipTestErr{})
}

func (t *t) Skipf(format string, args ...any) {
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *t) Skipped() bool {
	return t.skipped
}

func (t *t) TempDir() string {
	return ""
}

func (t *t) SetContext(key string, value any) {
	t.runContext[key] = value
}

func (t *t) Result() model.Result {
	if t.skipped {
		return model.ResultSkipped
	} else if t.passed {
		return model.ResultPassed
	}

	return model.ResultFailed
}

// skipTestErr is passed to panic() to signal
// that a test was skipped.
type skipTestErr struct{}

// failTestErr is passed to panic() to signal
// that a test has failed.
type failTestErr struct{}
