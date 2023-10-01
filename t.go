package handoff

import (
	"fmt"
	"strings"

	"github.com/raphi011/handoff/internal/model"
	"golang.org/x/exp/slog"
)

// make sure we adhere to the TB interface
var _ model.TB = &T{}

type T struct {
	suiteName      string
	testName       string
	logs           strings.Builder
	result         model.Result
	runtimeContext map[string]any
	cleanupFunc    func()
	softFailure    bool
}

func (t *T) Cleanup(c func()) {
	t.cleanupFunc = c
}

func (t *T) Error(args ...any) {
	t.result = model.ResultFailed
	t.Log(args...)
}

func (t *T) Errorf(format string, args ...any) {
	t.result = model.ResultFailed
	t.Logf(format, args...)
}

func (t *T) Fail() {
	t.result = model.ResultFailed
}

func (t *T) FailNow() {
	t.result = model.ResultFailed
	panic(failTestErr{})
}

func (t *T) Failed() bool {
	return t.result == model.ResultFailed
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
	t.logs.WriteString(fmt.Sprint(args...) + "\n")
}

func (t *T) Logf(format string, args ...any) {
	t.logs.WriteString(fmt.Sprintf(format, args...) + "\n")
}

func (t *T) Name() string {
	return t.testName
}

func (t *T) Setenv(key, value string) {
}

func (t *T) Skip(args ...any) {
	t.Log(args...)
	t.SkipNow()
}

func (t *T) SkipNow() {
	t.result = model.ResultSkipped
	panic(skipTestErr{})
}

func (t *T) Skipf(format string, args ...any) {
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *T) Skipped() bool {
	return t.result == model.ResultSkipped
}

func (t *T) TempDir() string {
	// TODO
	return ""
}

/* Handoff specific functions that are not part of the testing.TB interface */
/* ------------------------------------------------------------------------ */

func (t *T) GetContext(key string) any {
	return t.runtimeContext[key]
}

func (t *T) SoftFailure() {
	t.softFailure = true
}

func (t *T) SetContext(key string, value any) {
	t.runtimeContext[key] = value
}

func (t *T) Result() model.Result {
	if t.result == "" {
		return model.ResultPassed
	}

	return t.result
}

func (t *T) runTestCleanup() {
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
