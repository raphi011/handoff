package handoff

import (
	"fmt"
)

type T struct {
	name   string
	logs   []string
	passed bool
}

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
	Name  string `json:"name"`
	Tests map[string]TestFunc
}

type TestRun struct {
	ID         int32           `json:"id"`
	SuiteName  string          `json:"suiteName"`
	Results    []TestRunResult `json:"results"`
	TestFilter string          `json:"testFilter"`
	Tests      int             `json:"tests"`
	Passed     int             `json:"passed"`
	Failed     int             `json:"failed"`
}

type TestRunResult struct {
	Name   string   `json:"name"`
	Passed bool     `json:"passed"`
	Logs   []string `json:"logs"`
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
	panic("FailNow")
}

func (t *T) Failed() bool {
	return t.passed
}

func (t *T) Fatal(args ...any) {
	t.Error(args...)
	panic("Fatal")
}

func (t *T) Fatalf(format string, args ...any) {
	t.Errorf(format, args...)
	panic("Fatalf")
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

}

func (t *T) SkipNow() {

}

func (t *T) Skipf(format string, args ...any) {

}

func (t *T) Skipped() bool {
	return false
}

func (t *T) TempDir() string {
	return ""
}
