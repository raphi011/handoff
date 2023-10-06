# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and manually triggered e2e tests written in Go and is extensible through plugins.

## Example

Bootstrapping a server is simple, all you need to do is run this code:

```go
package main

func main() {
	h := handoff.New()
    h.Run()
}
```

To pass in test suites and scheduled runs you can do that by passing in `handoff.WithTestSuite` and `handoff.WithScheduledRun` options to `handoff.New()`.

Another way is to register them via `handoff.Register` before calling `handoff.New()`. This is especially convenient when you want to have your tests in the same repository as the system under test (SUT), which means they would be in a different repository (unless you have a monorepo). In this case the test package could register the tests in an init function like so:

```go
func init() {
    handoff.Register(ts, scheduledRuns)
}
```

and then all the handoff server needs to do is import the test package with a blank identifier:

```go
import _ "github.com/my-org/my-service/tests"
```

For examples see [./cmd/example-server-bootstrap/main.go] and [./internal/packagetestexample].

## Test best practices

* Pass in the test context for longer running operations and check if it was cancelled.
* Only log messages via t.Log/t.Logf as other log messages will not show up in the test logs.
* Make sure that code in `setup` is idempotent as it can run more than once.

## Planned features

- [ ] (Feature) Write a tool "transformcli" that uses go:generate and go/ast to transform handoff tests and suites to standard go tests (suite -> test with subtests + init and cleanup)
- [ ] (Feature) Automatic test run retries/backoff on failures
- [ ] (Feature) Configurable test run retention policy
- [ ] (Feature) Flaky test detection + metric
- [ ] (Feature) Add test-suite labels
- [ ] (Feature) Test suite namespaces
- [ ] (Feature) Asynchronous plugin hooks with callbacks for slow operations (e.g. http calls)
- [ ] (Technical) Comprehensive test suite
- [ ] (Plugin) Pagerduty - triger alerts/incidents on failed e2e tests
- [ ] (Plugin) Slack - send messages to slack channels when tests pass / fail
- [ ] (Plugin) Github - pr status checks
- [ ] (Plugin) Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
- [x] (Technical) Server configuration through either ENV vars or cli flags
- [x] (Technical) Continue test runs on service restart
- [x] (Technical) Graceful server shutdown
- [x] (Technical) Registering of `TestSuite`s and `ScheduledRun`s via imported packages
- [x] (Technical) SQLite Persistence layer
- [x] (Feature) Persist compressed test logs to save space
- [x] (Feature) Soft test fails that don't fail the entire testsuite. This can be used to help with the chicken/egg problem when you add new tests that target a new service version that is not deployed yet.
- [x] (Feature) Basic webui bundled in the service that shows test run results
- [x] (Feature) Start test runs via POST requests
- [x] (Feature) Test suite namespaces for grouping
- [x] (Feature) Write test suites with multiple tests written in Go
- [x] (Feature) Manual retrying of failed tests
- [x] (Feature) Skip individual tests by calling t.Skip() within a test
- [x] (Feature) Scheduled / recurring test runs (e.g. for soak tests)
- [x] (Feature) Skip test subsets via regex filters passed into a test run
- [x] (Feature) Support existing assertion libraries like stretch/testify
- [x] (Feature) Prometheus /metrics endpoint that exposes test metrics
- [x] (Feature) Basic support for plugins to hook into the test lifecycle

## Potential features

- [ ] (Technical) Limit the number of concurrent test runs via a configuration option
- [ ] (Technical) Websocket that streams test results (like test logs)
- [ ] (Technical) Authenticated HTTP requests through TLS client certificates
- [ ] (Feature) Grafana service dashboard template
- [ ] (Feature) Image for [helm chart](https://helm.sh/docs/topics/chart_tests/) tests for automated helm release rollbacks
- [ ] (Feature) Server mode + cli mode
- [ ] (Feature) Service dashboards that show information of services k8s resources running in a cluster and their test suite runs
- [ ] (Feature) Output go test json report
- [ ] (Feature) Create a helm chart that supports remote test debugging through dlv
- [ ] (Feature) Support running tests in languages other than go
- [ ] (Feature) k8s operator / CRDs to configure test runs & schedules
- [ ] (Feature) Opt-in test timeouts through t.Context and / or providing wrapped handoff functions ( e.g. http clients) to be used in tests  that implement test timeouts

## Open questions

- How to add test timeouts (it's impossible to externally stop goroutines running user provided functions)?

## Non goals

- Implement a new assertion library. We aim to be compatible with existing ones.

## Metrics

Metrics are exposed via the `/metrics` endpoint.

| Name                             | Type    | Description                                 | Labels                        |
| -------------------------------- | ------- | ------------------------------------------- | ----------------------------- |
| handoff_testsuites_running       | gauge   | The number of test suites currently running | namespace, suite_name         |
| handoff_testsuites_started_total | counter | The number of test suite runs started       | namespace, suite_name, result |
| handoff_tests_run_total          | counter | The number of tests run                     | namespace, suite_name, result |
