package handoff

import (
	"fmt"

	"github.com/raphi011/handoff/internal/model"
)

// make sure we adhere to the TB interface
var _ TB = &t{}

type t struct {
	name       string
	logs       string
	result     model.Result
	runContext map[string]any
}

func (t *t) Cleanup(func()) {
}

func (t *t) Error(args ...any) {
	t.result = model.ResultFailed
	t.Log(args...)
}

func (t *t) Errorf(format string, args ...any) {
	t.result = model.ResultFailed
	t.Logf(format, args...)
}

func (t *t) Fail() {
	t.result = model.ResultFailed
}

func (t *t) FailNow() {
	t.result = model.ResultFailed
	panic(failTestErr{})
}

func (t *t) Failed() bool {
	return t.result == model.ResultFailed
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
	t.logs += fmt.Sprint(args...) + "\n"
}

func (t *t) Logf(format string, args ...any) {
	t.logs += fmt.Sprintf(format, args...) + "\n"
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
	t.result = model.ResultSkipped
	panic(skipTestErr{})
}

func (t *t) Skipf(format string, args ...any) {
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *t) Skipped() bool {
	return t.result == model.ResultSkipped
}

func (t *t) TempDir() string {
	return ""
}

func (t *t) SetContext(key string, value any) {
	t.runContext[key] = value
}

func (t *t) Result() model.Result {
	if t.result == "" {
		return model.ResultPassed
	}

	return t.result
}

// skipTestErr is passed to panic() to signal
// that a test was skipped.
type skipTestErr struct{}

// failTestErr is passed to panic() to signal
// that a test has failed.
type failTestErr struct{}
