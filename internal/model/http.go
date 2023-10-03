package model

import "time"

type TestSuiteRunHTTP struct {
	// ID is the identifier of the test run.
	ID int `json:"id"`
	// SuiteName is the name of the test suite that is run.
	SuiteName string `json:"suiteName"`
	// Result is the outcome of the entire test suite run.
	Result Result `json:"result"`
	// TestFilter filters out a subset of the tests and skips the
	// remaining ones (not implemented yet).
	TestFilter string `json:"testFilter"`
	// Tests counts the total amount of tests in the suite.
	Tests int `json:"tests"`
	// Scheduled is the time when the test was triggered, e.g.
	// through a http call.
	Scheduled time.Time `json:"scheduled"`
	// Start is the time when the test run first started executing.
	Start time.Time `json:"start"`
	// End is the time when the test run finished executing.
	End time.Time `json:"end"`
	// DurationInMS is the test run execution times summed up.
	DurationInMS int64 `json:"durationInMs"`
	// SetupLogs are the logs that are written during the setup phase.
	SetupLogs string `json:"setupLogs"`
	// TriggeredBy denotes the origin of the test run, e.g. scheduled or via http call.
	TriggeredBy string `json:"triggeredBy"`
	// Environment is additional information on where the tests are run (e.g. cluster name).
	Environment string `json:"environment"`
	// TestResults contains the detailed test results of each test.
	TestResults []TestRunHTTP `json:"testResults"`
}

type TestRunHTTP struct {
	SuiteName  string `json:"suiteName"`
	SuiteRunID int    `json:"suiteRunId"`
	Name       string `json:"name"`
	// Result is the outcome of the test run.
	Result Result `json:"result"`
	// Test run attempt counter
	Attempt int `json:"attempt"`
	// SoftFailure if set to true, does not fail a test suite when the test run fails.
	SoftFailure bool `json:"softFailure"`
	// Logs contains log messages written by the test itself.
	Logs string `json:"logs"`
	// Start marks the start time of the test run.
	Start time.Time `json:"start"`
	// End marks the end time of the test run.
	End time.Time `json:"end"`
	// DurationInMS is the duration of the test run in milliseconds (end-start).
	DurationInMS int64 `json:"durationInMs"`
	// Context contains additional testrun specific information that is collected during and
	// after a test run either by the test itself (`t.SetContext`) or via plugins. This can
	// e.g. contain correlation ids or links to external services that may help debugging a test run
	// (among other things).
	Context TestContext `json:"context"`
}
