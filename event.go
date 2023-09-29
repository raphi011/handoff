package handoff

import (
	"fmt"
	"time"

	"github.com/raphi011/handoff/internal/model"
)

type testSuiteEvent interface {
	Apply(model.TestSuiteRun) model.TestSuiteRun
	RunID() int
	SuiteName() string
	LoadTestRuns() bool
}

type testRunEvent interface {
	Apply(model.TestRun) model.TestRun
	RunID() int
	SuiteName() string
	Attempt() int
	TestName() string
}

type testRunIdentifier struct {
	runID     int
	suiteName string
	attempt   int
	testName  string
}

func (e testRunIdentifier) SuiteName() string {
	return e.suiteName
}

func (e testRunIdentifier) RunID() int {
	return e.runID
}

func (e testRunIdentifier) Attempt() int {
	return e.attempt
}

func (e testRunIdentifier) TestName() string {
	return e.testName
}

type testSuiteRunEvent struct {
	runID        int
	suiteName    string
	loadTestRuns bool
}

func (e testSuiteRunEvent) SuiteName() string {
	return e.suiteName
}

func (e testSuiteRunEvent) RunID() int {
	return e.runID
}

func (e testSuiteRunEvent) LoadTestRuns() bool {
	return e.loadTestRuns
}

type testSuiteRunStartedEvent struct {
	testSuiteRunEvent
	start time.Time
}

func (e testSuiteRunStartedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	timeNotSet := time.Time{}

	// only set start time if it wasn't set before. This
	// is possible if we resume a paused test suite run.
	if ts.Start == timeNotSet {
		ts.Start = e.start
	}

	return ts
}

type testSuiteRunFinishedEvent struct {
	testSuiteRunEvent
	end time.Time
}

func (e testSuiteRunFinishedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.End = e.end

	result := model.ResultPassed

	for _, r := range ts.TestResults {
		// todo: only use latest attempts of each test
		ts.DurationInMS += r.DurationInMS

		if r.Result == model.ResultFailed {
			result = model.ResultFailed
			break
		}
	}
	ts.Result = result

	return ts
}

type testSuiteRunSetupFailedEvent struct {
	testSuiteRunEvent
	end time.Time
	err error
}

func (e testSuiteRunSetupFailedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.Result = model.ResultSetupFailed
	ts.End = e.end

	return ts
}

type testFinishedEvent struct {
	testRunIdentifier
	start       time.Time
	end         time.Time
	result      model.Result
	recovery    any
	logs        string
	testContext model.TestContext
}

func (e testFinishedEvent) Apply(ts model.TestRun) model.TestRun {
	logs := e.logs

	result := e.result

	if e.recovery != nil && e.result != model.ResultSkipped {
		if _, ok := e.recovery.(failTestErr); !ok {
			// this is an unexpected panic (does not originate from handoff)
			logs += fmt.Sprintf("%v\n", e.recovery)
			result = model.ResultFailed
		}
	}

	ts.Start = e.start
	ts.End = e.end
	ts.DurationInMS = e.end.Sub(e.start).Milliseconds()
	ts.Result = result
	ts.Logs = logs
	ts.Context = e.testContext

	return ts
}
