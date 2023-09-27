package handoff

import (
	"fmt"
	"time"

	"github.com/raphi011/handoff/internal/model"
)

type event interface {
	Apply(model.TestSuiteRun) model.TestSuiteRun
	RunID() int
	SuiteName() string
}

type testRunIdentifier struct {
	runID     int
	suiteName string
}

func (e testRunIdentifier) SuiteName() string {
	return e.suiteName
}

func (e testRunIdentifier) RunID() int {
	return e.runID
}

type testRunStartedEvent struct {
	testRunIdentifier
	start time.Time
}

func (e testRunStartedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	timeNotSet := time.Time{}

	// only set start time if it wasn't set before. This
	// is possible if we resume a paused test suite run.
	if ts.Start == timeNotSet {
		ts.Start = e.start
	}

	return ts
}

type testRunFinishedEvent struct {
	testRunIdentifier
	end time.Time
	// skipped is the # of tests skipped by the run TestFilter
	skipped int
}

func (e testRunFinishedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.End = e.end

	// todo: make sure test results contains every result
	for _, r := range ts.TestResults {
		ts.DurationInMS += r.DurationInMS
	}

	// add to skipped because each test can also call t.Skip()
	ts.Skipped += e.skipped

	result := model.ResultPassed

	if ts.Failed > 0 {
		result = model.ResultFailed
	}

	ts.Result = result

	return ts
}

type testRunSetupFailedEvent struct {
	testRunIdentifier
	end time.Time
	err error
}

func (e testRunSetupFailedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.Result = model.ResultSetupFailed
	ts.End = e.end

	return ts
}

type testFinishedEvent struct {
	testRunIdentifier
	start    time.Time
	end      time.Time
	result   model.Result
	testName string
	recovery any
	logs     string
}

func (e testFinishedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	logs := e.logs

	result := e.result

	if e.recovery != nil && e.result != model.ResultSkipped {
		if _, ok := e.recovery.(failTestErr); !ok {
			// this is an unexpected panic (does not originate from handoff)
			logs += fmt.Sprintf("%v\n", e.recovery)
			result = model.ResultFailed
		}
	}

	switch e.result {
	case model.ResultSkipped:
		ts.Skipped++
	case model.ResultPassed:
		ts.Passed++
	case model.ResultFailed:
		ts.Failed++
	}

	ts.TestResults = append(ts.TestResults, model.TestRun{
		Name:         e.testName,
		Result:       result,
		Logs:         logs,
		Start:        e.start,
		End:          e.end,
		DurationInMS: e.end.Sub(e.start).Milliseconds(),
	})

	return ts
}
