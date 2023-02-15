package handoff

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
