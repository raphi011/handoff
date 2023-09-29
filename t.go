package handoff

import (
	"fmt"
	"strings"

	"github.com/raphi011/handoff/internal/model"
	"golang.org/x/exp/slog"
)

// make sure we adhere to the TB interface
var _ TB = &t{}

type t struct {
	suiteName      string
	testName       string
	logs           strings.Builder
	result         model.Result
	runtimeContext map[string]any
	cleanupFunc    func()
}

func (t *t) Cleanup(c func()) {
	t.cleanupFunc = c
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
	t.logs.WriteString(fmt.Sprint(args...) + "\n")
}

func (t *t) Logf(format string, args ...any) {
	t.logs.WriteString(fmt.Sprintf(format, args...) + "\n")
}

func (t *t) Name() string {
	return t.testName
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
	// TODO
	return ""
}

/* Handoff specific functions that are not part of the testing.TB interface */
/* ------------------------------------------------------------------------ */

func (t *t) GetContext(key string) any {
	return t.runtimeContext[key]
}

func (t *t) SetContext(key string, value any) {
	t.runtimeContext[key] = value
}

func (t *t) Result() model.Result {
	if t.result == "" {
		return model.ResultPassed
	}

	return t.result
}

func (t *t) runTestCleanup() {
	if t.cleanupFunc == nil {
		return
	}

	defer func() {
		err := recover()

		if err != nil {
			slog.Warn("cleanup func panic'd", "error", err, "suite-name", t.suiteName, "test-name", t.testName)
		}
	}()

	t.cleanupFunc()
}

// skipTestErr is passed to panic() to signal
// that a test was skipped.
type skipTestErr struct{}

// failTestErr is passed to panic() to signal
// that a test has failed.
type failTestErr struct{}
