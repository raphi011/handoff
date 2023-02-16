package handoff

type Plugin interface {
	Name() string
	Init() error
	Stop() error
}

type TestStartedListener interface {
	TestStarted(suite TestSuite, run TestRun, testName string)
}

type TestFinishedListener interface {
	TestFinished(suite TestSuite, run TestRun, testName string, context map[string]any)
}

type TestSuiteStartedListener interface {
	TestSuiteStarted()
}

type TestSuiteFinishedListener interface {
	TestSuiteFinished()
}

// PagerDutyPlugin supports creating and resolving incidents when
// testsuites fail.
type PagerDutyPlugin struct {
}

// GithubPlugin supports running testsuites on PRs.
type GithubPlugin struct {
}

// SlackPlugin supports sending messages to slack channels that inform on
// test runs.
type SlackPlugin struct {
}

// LogstashPlugin supports fetching logs created by test runs.
type LogstashPlugin struct {
}
