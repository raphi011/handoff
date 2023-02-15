package handoff

import (
	"fmt"
	"time"
)

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
	Scheduled   time.Time
	TriggeredBy string
}

func (e TestRunStarted) Apply(ts TestRun) TestRun {
	ts.ID = e.runID
	ts.SuiteName = e.suiteName
	ts.TestResults = []TestRunResult{}
	ts.Scheduled = e.Scheduled
	ts.SetupLogs = []string{}

	return ts
}

type TestRunFinished struct {
	TestRunIdentifier
	start time.Time
	end   time.Time
}

func (e TestRunFinished) Apply(ts TestRun) TestRun {
	ts.Start = e.start
	ts.End = e.end
	ts.DurationInMS = e.end.Sub(e.start).Milliseconds()

	result := ResultPassed

	if ts.Failed > 0 {
		result = ResultFailed
	}

	ts.Result = result

	return ts
}

type TestRunSetupFailed struct {
	TestRunIdentifier
	start time.Time
	end   time.Time
	err   error
}

func (e TestRunSetupFailed) Apply(ts TestRun) TestRun {
	ts.Result = ResultSetupFailed
	ts.Start = e.start
	ts.End = e.end
	ts.DurationInMS = e.end.Sub(e.start).Milliseconds()

	return ts
}

type TestFinished struct {
	TestRunIdentifier
	start    time.Time
	end      time.Time
	skipped  bool
	testName string
	recovery any
	passed   bool
	logs     []string
}

func (e TestFinished) Apply(ts TestRun) TestRun {
	passed := e.passed
	logs := e.logs

	if e.recovery != nil && !e.skipped {
		if _, ok := e.recovery.(failTestErr); !ok {
			// this looks like an unexpected panic (does not originate
			// from handoff), therefor log it
			logs = append(logs, fmt.Sprintf("%v", e.recovery))
		}

		passed = false
	}

	if e.skipped {
		ts.Skipped++
	} else if passed {
		ts.Passed++
	} else {
		ts.Failed++
	}

	ts.Tests++

	ts.TestResults = append(ts.TestResults, TestRunResult{
		Name:         e.testName,
		Passed:       passed,
		Logs:         logs,
		Skipped:      e.skipped,
		Start:        e.start,
		End:          e.end,
		DurationInMS: e.end.Sub(e.start).Milliseconds(),
	})

	return ts
}
