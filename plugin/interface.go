package plugin

import (
	"github.com/raphi011/handoff/internal/model"
)

type TestStartedListener interface {
	TestStarted(suite model.TestSuite, run model.TestSuiteRun, testName string)
}

type TestFinishedListener interface {
	TestFinished(suite model.TestSuite, run model.TestSuiteRun, testName string, context map[string]any)
}

type TestSuiteStartedListener interface {
	TestSuiteStarted()
}

type TestSuiteFinishedListener interface {
	TestSuiteFinished()
}
