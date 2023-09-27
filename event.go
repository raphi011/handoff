package handoff

import (
	"fmt"
	"regexp"
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
	scheduled   time.Time
	triggeredBy string
	testFilter  *regexp.Regexp
	tests       int
	environment string
}

func (e testRunStartedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.ID = e.runID
	ts.SuiteName = e.suiteName
	ts.TestResults = []model.TestRun{}
	ts.Scheduled = e.scheduled
	ts.TestFilterRegex = e.testFilter
	ts.Tests = e.tests
	ts.Result = model.ResultPending
	ts.Environment = e.environment

	if e.testFilter != nil {
		ts.TestFilter = e.testFilter.String()
	}

	return ts
}

type testRunFinishedEvent struct {
	testRunIdentifier
	start time.Time
	end   time.Time
	// skipped is the # of tests skipped by the run TestFilter
	skipped int
}

func (e testRunFinishedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.Start = e.start
	ts.End = e.end
	ts.DurationInMS = e.end.Sub(e.start).Milliseconds()
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
	start time.Time
	end   time.Time
	err   error
}

func (e testRunSetupFailedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.Result = model.ResultSetupFailed
	ts.Start = e.start
	ts.End = e.end
	ts.DurationInMS = e.end.Sub(e.start).Milliseconds()

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
