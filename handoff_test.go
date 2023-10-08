package handoff_test

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/raphi011/handoff"
	"github.com/raphi011/handoff/client"
	"github.com/raphi011/handoff/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSuiteWithFailingTestShouldFailTheRun(t *testing.T) {
	t.Parallel()

	suiteName := "failing"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultFailed)
}

func TestSuiteWithSoftFailShouldNotFailTheRun(t *testing.T) {
	t.Parallel()

	suiteName := "soft-fail"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultPassed)
}

func TestSuiteWithNoFailingTestsShouldSucceed(t *testing.T) {
	t.Parallel()

	suiteName := "succeed"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultPassed)
}

func TestSuiteNeedsRetrySucceedsOnTheSecondAttempt(t *testing.T) {
	t.Parallel()

	suiteName := "needs-retry"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	tsr = te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultPassed)

	tr := latestTestAttempt(t, tsr, "Retry")
	assert.Equal(t, 2, tr.Attempt, "expected 2 test run attempts")
	assert.Equal(t, model.ResultPassed, tr.Result, "expected test run to have passed")
}

func TestSuiteWithFailingSetupSkipsTestsAndFails(t *testing.T) {
	t.Parallel()

	suiteName := "failing-setup"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	tsr = te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultFailed)
	assert.Equal(t, model.ResultFailed, tsr.Result, "expected test suite run to fail")
	assert.Equal(t, "setup failed: error", tsr.SetupLogs, "expected test suite run setup logs to contain err")

	tr := latestTestAttempt(t, tsr, "Success")
	assert.Equal(t, 1, tr.Attempt, "expected 1 test run attempt")
	assert.Equal(t, model.ResultSkipped, tr.Result, "expected test run to have been skipped")
	assert.Equal(t, "test suite run setup failed: skipped", tr.Logs, "expected test run to contain setup failed log")
}

func TestSuiteWithPanicingSetupSkipsTestsAndFails(t *testing.T) {
	t.Parallel()

	suiteName := "panicing-setup"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	tsr = te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultFailed)
	assert.Equal(t, model.ResultFailed, tsr.Result, "expected test suite run to fail")
	assert.Equal(t, "setup failed: panic", tsr.SetupLogs, "expected test suite run setup logs to contain panic")

	tr := latestTestAttempt(t, tsr, "Success")
	assert.Equal(t, 1, tr.Attempt, "expected 1 test run attempt")
	assert.Equal(t, model.ResultSkipped, tr.Result, "expected test run to have been skipped")
	assert.Equal(t, "test suite run setup failed: skipped", tr.Logs, "expected test run to contain setup failed log")
}

func TestSuiteRegisteredByExternalPackage(t *testing.T) {
	t.Parallel()

	suiteName := "external-suite-succeed"

	tsr := te.createNewTestSuiteRun(t, suiteName)

	te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultPassed)
}

func TestTestSuiteValidation(t *testing.T) {
	t.Parallel()

	os.Args = []string{}

	h := handoff.New(handoff.WithTestSuite(handoff.TestSuite{
		Name: "",
		Tests: []model.TestFunc{
			Success,
		},
	}))

	err := h.Run([]string{})
	assert.Error(t, err, "Passing in a test suite without a name should fail")

	h = handoff.New(handoff.WithTestSuite(handoff.TestSuite{
		Name: "success",
	}))

	err = h.Run([]string{})
	assert.Error(t, err, "Passing in a test suite without any tests should fail")

	h = handoff.New(handoff.WithTestSuite(handoff.TestSuite{
		Name: "success",
		Tests: []model.TestFunc{
			Success, Success,
		},
	}), handoff.WithTestSuite(handoff.TestSuite{
		Name: "success",
		Tests: []model.TestFunc{
			Success, Success,
		},
	}))

	err = h.Run([]string{})
	assert.Error(t, err, "Passing in multiple test suite with the same name should fail")
}

func TestShutdownSucceeds(t *testing.T) {
	t.Parallel()

	i := handoffInstance([]handoff.TestSuite{
		{
			Name: "success",
			Tests: []model.TestFunc{
				Sleep(2 * time.Second),
			},
		}},
		[]handoff.ScheduledRun{
			{
				TestSuiteName: "success",
				Schedule:      "@every 1.5s",
			}},
		[]string{"handoff-test", "-p", "0", "-d", ""},
	)

	// make sure the scheduled run has started
	time.Sleep(2 * time.Second)

	err := i.h.Shutdown()

	assert.NoError(t, err, "service shutdown should succeed")
}

func TestScheduledRunWithTestFilter(t *testing.T) {
	t.Parallel()

	suiteName := "success"
	filteredTestName := "LogAttempt"

	suites := []handoff.TestSuite{{
		Name:  suiteName,
		Tests: []model.TestFunc{Success, LogAttempt},
	}}

	scheduledRuns := []handoff.ScheduledRun{{
		TestSuiteName: suiteName,
		Schedule:      "@every 1.5s",
		TestFilter:    regexp.MustCompile(filteredTestName),
	}}

	i := handoffInstance(suites, scheduledRuns, []string{"handoff-test", "-p", "0", "-d", ""})

	// make sure the scheduled run has started
	time.Sleep(2 * time.Second)

	tsr := i.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, 1, model.ResultPassed)

	tr := latestTestAttempt(t, tsr, filteredTestName)
	assert.Equal(t, model.ResultPassed, tr.Result)

	tr = latestTestAttempt(t, tsr, "Success")
	assert.Equal(t, model.ResultSkipped, tr.Result)

	err := i.h.Shutdown()
	assert.NoError(t, err, "service shutdown should succeed")
}

func TestSuiteRunWithUnknownSuiteShouldFailSuiteNotFoundReturns404(t *testing.T) {
	t.Parallel()

	_, err := te.client.CreateTestSuiteRun(context.Background(), "not-found", nil)

	var reqError client.RequestError

	assert.ErrorAs(t, err, &reqError, "expected error of type RequestError")
	assert.Equal(t, http.StatusNotFound, reqError.ResponseCode, "expected response code status not found")
}

func TestCreateTestSuiteRunShouldSucceed(t *testing.T) {
	t.Parallel()

	_, err := te.client.CreateTestSuiteRun(context.Background(), "succeed", nil)
	assert.NoError(t, err, "create test suite run should not fail")
}

func TestGetTestRunShouldSucceed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	suiteName := "needs-retry"

	tsr := te.createNewTestSuiteRun(t, suiteName)
	te.waitForTestSuiteRunWithResult(t, defaultTimeout, "needs-retry", tsr.ID, model.ResultPassed)

	tr, err := te.client.GetTestRun(ctx, suiteName, tsr.ID, "Retry")
	assert.NoError(t, err, "get test run should not fail")
	assert.Len(t, tr, 2, "expected two test runs")
	assert.Equal(t, 1, tr[0].Attempt, "expected the first test run attempt")
	assert.Equal(t, 2, tr[1].Attempt, "expected the second test run attempt")
}

func TestCreateNewTestSuiteRunWithFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	suiteName := "multiple-tests"

	tsr, err := te.client.CreateTestSuiteRun(ctx, suiteName, regexp.MustCompile("Success|Retry"))
	assert.NoError(t, err, "creating test suite run should succeed")

	tsr = te.waitForTestSuiteRunWithResult(t, defaultTimeout, suiteName, tsr.ID, model.ResultPassed)
	assert.Len(t, tsr.TestResults, 4, "expected 4 test runs")

	flakyTest := latestTestAttempt(t, tsr, "Flaky")
	assert.Equal(t, model.ResultSkipped, flakyTest.Result)

	successTest := latestTestAttempt(t, tsr, "Success")
	assert.Equal(t, model.ResultPassed, successTest.Result)

	retryTest := latestTestAttempt(t, tsr, "Retry")
	assert.Equal(t, model.ResultPassed, retryTest.Result)
}

func TestResumePendingTests(t *testing.T) {
	// TODO
}
