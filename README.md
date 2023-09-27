# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and manually triggered e2e tests written in Go.

## Example

See [the example app](./cmd/example/main.go).

## Planned features

- [ ] (Feature) Server mode + cli mode
- [ ] (Feature) Image for [helm chart](https://helm.sh/docs/topics/chart_tests/) tests for automated helm release rollbacks
- [ ] (Feature) Use go:generate to generate go tests from handoff tests to execute with standard go tooling
- [ ] (Feature) Output go test json report
- [ ] (Feature) Test run backoff on failures
- [ ] (Feature) Configurable test run retention policy
- [ ] (Feature) Soft test fails that don't fail the testsuite
- [ ] (Feature) Service dashboards that show information of services k8s resources running in a cluster and their test suite runs
- [ ] (Technical) Websocket that returns realtime test results (including test logs)
- [ ] (Technical) Authenticated HTTP requests through TLS client certificates
- [ ] (Technical) Continue test runs on service restart
- [ ] (Plugin) Pagerduty - failed e2e tests can triger alerts / incidents
- [ ] (Plugin) Slack - send messages to slack channels when tests pass / fail
- [ ] (Plugin) Github - pr status checks
- [ ] (Plugin) Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
- [x] (Technical) Graceful server shutdown
- [x] (Technical) Persistence layer
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

- [ ] (Feature) Create a helm chart that supports remote test debugging through dlv
- [ ] (Feature) Support running tests in languages other than go and collect test results
- [ ] (Feature) Flaky test detection
- [ ] (Feature) k8s operator / CRDs to configure test runs & schedules

## Non goals

- Implement a new assertion library. We should aim to be compatible with existing ones.

## Technical limitations

- No way to set a timeout for test functions

## Metrics

Metrics are exposed via the `/metrics` endpoint.

| Name                         | Type    | Description                                                 | Labels                                 |
| ---------------------------- | ------- | ----------------------------------------------------------- | -------------------------------------- |
| handoff_testsuites_running   | gauge   | The number of test suites currently running                 | associated_service, suite_name         |
| handoff_testsuites_run_total | counter | The number of test suites run since the service was started | associated_service, suite_name, result |
| handoff_tests_run_total      | counter | The number of tests run since the service was started       | associated_service, suite_name, result |

