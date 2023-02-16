# Handoff

Handoff is a library that allows you to bootstrap a server that runs scheduled and triggered e2e tests written in Go.


## Example

See [the example app](./cmd/example/main.go).

## Features

- [x] Start test runs via POST requests
- [x] Write test suites with multiple tests written in Go
- [x] Skip tests
- [x] Scheduled / recurring test runs (e.g. soak tests)
- [x] Execute test subsets via filters
- [x] Support existing assertion libraries like stretch/testify
- [ ] Export prometheus test metrics
- [ ] Test run retention policy
- [ ] Webui that shows test runs
- [ ] k8s CRDs to configure test runs
- [ ] output go test json report
- [ ] Websocket that returns realtime test results (including test logs)
- Plugins
  - [ ] Pagerduty - failed e2e tests can triger alerts / incidents
  - [ ] Slacks - send messages to slack channels when tests pass / fail
  - [ ] Github - pr status checks
  - [ ] Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
