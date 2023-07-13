# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and triggered e2e tests written in Go.

## Example

See [the example app](./cmd/example/main.go).

## Features

- [x] Start test runs via POST requests
- [x] Write test suites with multiple tests written in Go
- [x] Skip individual tests by calling t.Skip() within a test
- [x] Scheduled / recurring test runs (e.g. soak tests)
- [x] Skip test subsets via regex filters passed into a test run
- [x] Support existing assertion libraries like stretch/testify
- [x] Export prometheus test metrics
- [x] Basic support for plugins to hook into the test lifecycle
- [ ] Webui that shows test run results
- [ ] flaky test detection
- [ ] server mode + cli mode
- [ ] image for [helm chart](https://helm.sh/docs/topics/chart_tests/) tests for automated helm release rollbacks
- [ ] CLI to run tests
- [ ] Authenticated HTTP requests through TLS client certificates
- [ ] Use go:generate to generate go tests from handoff tests to execute with standard go tooling
- [ ] k8s operator / CRDs to configure test runs & schedules
- [ ] Output go test json report
- [ ] test run backoff on failures
- [ ] Websocket that returns realtime test results (including test logs)
- [ ] Persistence layer (e.g. sqlite)
- [ ] Configurable test run retention policy
- [ ] Support running tests in languages other than go and collect text results
- [ ] Service dashboards that show information of services k8s resources running in a cluster and their test suite runs
- Plugins
  - [ ] Pagerduty - failed e2e tests can triger alerts / incidents
  - [ ] Slack - send messages to slack channels when tests pass / fail
  - [ ] Github - pr status checks
  - [ ] Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids

## Non goals

- Implement a new assertion library, rather be compatible with existing ones

## Metrics

Metrics are exposed via the `/metrics` endpoint.

| Name                         | Type    | Description                                                 | Labels                                 |
| ---------------------------- | ------- | ----------------------------------------------------------- | -------------------------------------- |
| handoff_testsuites_running   | gauge   | The number of test suites currently running                 | associated_service, suite_name         |
| handoff_testsuites_run_total | counter | The number of test suites run since the service was started | associated_service, suite_name, result |
| handoff_tests_run_total      | counter | The number of tests run since the service was started       | associated_service, suite_name, result |

