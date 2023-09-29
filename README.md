# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and manually triggered e2e tests written in Go and is extensible through plugins.

## Example

See [the example app](./cmd/example/main.go).

## Planned features

- [ ] (Feature) Image for [helm chart](https://helm.sh/docs/topics/chart_tests/) tests for automated helm release rollbacks
- [ ] (Feature) Write a tool "transformcli" that uses go:generate and go/ast to transform handoff tests and suites to standard go tests (suite -> test with subtests + init and cleanup)
- [ ] (Feature) Automatic test run retries/backoff on failures
- [ ] (Feature) Configurable test run retention policy
- [ ] (Feature) Manual retrying of failed tests
- [ ] (Feature) Grafana service dashboard template
- [ ] (Feature) Soft test fails that don't fail the entire testsuite. This can be used to help with the chicken/egg problem when you add new tests that target a new service version that is not deployed yet.
- [ ] (Feature) Flaky test detection + metric
- [ ] (Feature) Add test-suite labels (e.g. instead of "associated service", "team name" attributes)
- [ ] (Feature) Add support for async plugins in case they need to do slow operations such as http calls
- [ ] (Feature) Configuration through either ENV vars or cli flags
- [ ] (Feature) Asynchronous plugin hooks with callbacks for slow operations (e.g. http calls)
- [ ] (Technical) Limit the number of concurrent test runs via a configuration option
- [ ] (Technical) Websocket that streams test results (like test logs)
- [ ] (Technical) Authenticated HTTP requests through TLS client certificates
- [ ] (Technical) Continue test runs on service restart
- [ ] (Plugin) Pagerduty - triger alerts/incidents on failed e2e tests
- [ ] (Plugin) Slack - send messages to slack channels when tests pass / fail
- [ ] (Plugin) Github - pr status checks
- [ ] (Plugin) Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
- [x] (Technical) Graceful server shutdown
- [x] (Technical) SQLite Persistence layer
- [x] (Feature) Basic webui that shows test run results
- [x] (Feature) Start test runs via POST requests
- [x] (Feature) Write test suites with multiple tests written in Go
- [x] (Feature) Skip individual tests by calling t.Skip() within a test
- [x] (Feature) Scheduled / recurring test runs (e.g. for soak tests)
- [x] (Feature) Skip test subsets via regex filters passed into a test run
- [x] (Feature) Support existing assertion libraries like stretch/testify
- [x] (Feature) Prometheus /metrics endpoint that exposes test metrics
- [x] (Feature) Basic support for plugins to hook into the test lifecycle

## Potential features

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

| Name                         | Type    | Description                                                 | Labels                                 |
| ---------------------------- | ------- | ----------------------------------------------------------- | -------------------------------------- |
| handoff_testsuites_running   | gauge   | The number of test suites currently running                 | associated_service, suite_name         |
| handoff_testsuites_run_total | counter | The number of test suites run since the service was started | associated_service, suite_name, result |
| handoff_tests_run_total      | counter | The number of tests run since the service was started       | associated_service, suite_name, result |
