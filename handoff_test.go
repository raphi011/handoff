package handoff_test

import (
	"testing"
	"time"

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

	tr := latestTestAttempt(t, tsr, "RetryOnce")

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

func TestScheduledRunIsCreated(t *testing.T) {
	t.Parallel()

	suiteName := "plugin-scheduled-test"

	te.waitForTestSuiteRunWithResult(t, 5*time.Second, suiteName, 1, model.ResultPassed)
}

