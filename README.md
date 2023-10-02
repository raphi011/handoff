# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and manually triggered e2e tests written in Go and is extensible through plugins.

## Example

There are several ways to setup a handoff server: 

1. Build the `./cmd/handoff` binary and your custom test suite libraries (see `./cmd/example-test-library`) and pass in the libraries via the commandline:

```sh
go install github.com/raphi011/handoff/cmd/handoff
go build -buildmode plugin -o example-test-library ./cmd/example-test-library
handoff -t ./example-test-library
```

A test suite library needs to include a public function with signature `func Handoff() ([]handoff.TestSuite, []handoff.ScheduledRun)`.

2. Bootstrap a handoff server yourself by importing the library (see `./cmd/example-server-bootstrap`) and calling `handoff.New().Run()` with tests 
defined in the same repository and passed in via the `handoff.WithTestSuite()` option.

3. Bootstrap a handoff server and import `TestSuite`s from external packages. This is a handy approach if multiple teams use the same server to run their tests as the tests can live in separate codebases.

4. A combination of 2 & 3.

## Planned features

- [ ] (Feature) Write a tool "transformcli" that uses go:generate and go/ast to transform handoff tests and suites to standard go tests (suite -> test with subtests + init and cleanup)
- [ ] (Feature) Automatic test run retries/backoff on failures
- [ ] (Feature) Configurable test run retention policy
- [ ] (Feature) Flaky test detection + metric
- [ ] (Feature) Add test-suite labels
- [ ] (Feature) Asynchronous plugin hooks with callbacks for slow operations (e.g. http calls)
- [ ] (Technical) Comprehensive test suite
- [ ] (Plugin) Pagerduty - triger alerts/incidents on failed e2e tests
- [ ] (Plugin) Slack - send messages to slack channels when tests pass / fail
- [ ] (Plugin) Github - pr status checks
- [ ] (Plugin) Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
- [x] (Technical) Server configuration through either ENV vars or cli flags
- [x] (Technical) Continue test runs on service restart
- [x] (Technical) Graceful server shutdown
- [x] (Technical) Loading of `TestSuite`s via shared libraries.
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
