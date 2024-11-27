# Features

A list of planned / implemented features.

## Roadmap to v0.1

- [ ] [Nice Web UI that supports the most important functionality](#basic-webui)
- [ ] [Elasticsearch integration](#elasticsearch-hook)
- [ ] [Slack integration](#slack-hook)
- [ ] [CLI tool](#cli-tool)
- [ ] [Documentation site](#documentation)
- [ ] [Helm Chart](#helm-chart)
- [ ] [Scheduled Tests](#scheduled-tests)

## Planned

### Basic WebUI

* List all test suites
* List all test runs and the individual tests
* Create new test runs
* See test run schedules

## Slack hook

Configurable test run status updates that are pushed to slack channels / users.

### Transformcli

Write a tool "transformcli" that uses go:generate and go/ast to transform handoff tests and suites to standard go tests (suite -> test with subtests + init and cleanup)

### CLI Tool

A tool that implements handoff's api and makes it easy to

* Trigger new test runs, via the CLI
* Fetch the status of a test
* Export results to json
* ...

### Documentation

Hosted documentation on how to use handoff, e.g. via readthedocs.com.

### Helm Chart

A helm chart that makes it easy to deploy handoff to kubernetes.

### Users

Teams (assigned to test suites)
Favorite test suites (UI)
authentication providers (LDAP, Oauth2, ...)

### Testsuite Metadata

Test Suite Metadata (desription in markdown, links to e.g. github, documentation, handbook, ...) shown in UI.

### Scheduled Tests

The ability to run test suites in a repeated fashion via cron job expressions. Optionally set a max amount of runs.

### Elasticsearch hook

Find and fetch log statements from the systems under test (SUT) via correlation ids.

### Pagerduty integration

Trigger alerts/incidents on failed e2e tests.

### Testrun metrics

Prometheus /metrics endpoint that exposes test run metrics.

### Improved UI

* Dashboard UI that shows handoff statistics, running tests, resource usage (cpu, memory, active go routines...) etc

### Email Notifications

Test run status updates via email.

Idea: figure out email from committer via commit that is part of a pr that triggered a test.

### Jira Integration

Add test run results to a related ticket.

### Headless Server

Allow running handoff as a cli tool without a webui to execute tests in a CI environment.

## Backlog

- [ ] (Feature) Opt-in test timeouts through t.Context and / or providing wrapped handoff functions (e.g. http clients) to be used in tests  that implement test timeouts
- [ ] (Feature) Add test-suite labels
- [ ] (Feature) Allow running of handoff as headless/cli mode (without http server) that returns a code != 0 if a test has failed (e.g. in github actions CI)
- [ ] (Feature) Add an option to the helm chart to support remote debugging through dlv
- [ ] (Feature) Image for [helm chart](https://helm.sh/docs/topics/chart_tests/) tests for automated helm release rollbacks
- [ ] (Feature) Test suite namespaces for grouping
- [ ] (Plugin) Github - add test results as PR comments after it was merged
- [ ] (Plugin) Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids
- [x] (Feature) Soft test fails that don't fail the entire testsuite. This can be used to help with the chicken/egg problem when you add new tests that target a new service version that is not deployed yet.
- [x] (Feature) Configurable test run retention policy (TTL)

## Potential features

* [ ] (Technical) Websocket that streams test results (like test logs) - this could be used by the cli tool to get live updates on running tests
* [ ] (Technical) Authenticated HTTP requests through TLS client certificates
* [ ] (Feature) Grafana service dashboard template
* [ ] (Feature) Service dashboards that show information of services k8s resources running in a cluster and their test suite runs
* [ ] (Feature) Output go test json report
* [ ] (Feature) Support running tests in languages other than go
* [ ] (Feature) k8s operator / CRDs to configure test runs & schedules (we probably don't need this)


## dumping ground

- [ ] (Feature) Automatic test run retries/backoff on failures
- [ ] (Technical) Run tests in parallel
- [ ] (Technical) Index data (e.g. with github.com/blevesearch/bleve) to be able to query test results.
