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
}

func (e testRunStartedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	ts.ID = e.runID
	ts.SuiteName = e.suiteName
	ts.TestResults = []model.TestRun{}
	ts.Scheduled = e.scheduled
	ts.TestFilterRegex = e.testFilter
	ts.Tests = e.tests
	ts.Result = model.ResultPending

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
	skipped  bool
	testName string
	recovery any
	passed   bool
	logs     string
}

func (e testFinishedEvent) Apply(ts model.TestSuiteRun) model.TestSuiteRun {
	passed := e.passed
	logs := e.logs

	if e.recovery != nil && !e.skipped {
		if _, ok := e.recovery.(failTestErr); !ok {
			// this is an unexpected panic (does not originate from handoff)
			logs += fmt.Sprintf("%v\n", e.recovery)
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

	ts.TestResults = append(ts.TestResults, model.TestRun{
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
