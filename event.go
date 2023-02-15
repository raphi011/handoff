package handoff

import "fmt"

type Event interface {
	Apply(TestRun) TestRun
	RunID() int32
	SuiteName() string
}

type TestRunIdentifier struct {
	runID     int32
	suiteName string
}

func (e TestRunIdentifier) SuiteName() string {
	return e.suiteName
}

func (e TestRunIdentifier) RunID() int32 {
	return e.runID
}

type TestRunStarted struct {
	TestRunIdentifier
}

func (e TestRunStarted) Apply(ts TestRun) TestRun {
	ts.ID = e.runID
	ts.SuiteName = e.suiteName
	ts.Results = []TestRunResult{}

	return ts
}

type TestFinished struct {
	TestRunIdentifier
	testName string
	recovery any
	passed   bool
	logs     []string
}

func (e TestFinished) Apply(ts TestRun) TestRun {
	passed := e.passed
	logs := e.logs

	if e.recovery != nil {
		logs = append(logs, fmt.Sprintf("%v", e.recovery))
		passed = false
	}

	if passed {
		ts.Passed++
	} else {
		ts.Failed++
	}
	ts.Tests++

	ts.Results = append(ts.Results, TestRunResult{
		Name:   e.testName,
		Passed: passed,
		Logs:   logs,
	})

	return ts
}
