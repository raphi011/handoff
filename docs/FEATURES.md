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

## List of potential features

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

Dashboard UI that shows handoff statistics, running tests, resource usage (cpu, memory, active go routines...) etc

### Email Notifications

Test run status updates via email.

Idea: figure out email from committer via commit that is part of a pr that triggered a test.

### Jira Integration

Add test run results to a related ticket.

### Headless Server

Allow running handoff as a cli tool in headless mode (without the web server) to e.g. run tests in a CI environment and return a code != 0 if the tests have failed.

### Automatic Pprof Profiling

A test can automatically fetch pprof profiles (go only) from the SUT while the test is running and them for later inspection.

### Test Timeouts / Test cancellation

Opt-in test timeouts through t.Context and / or providing wrapped handoff functions (e.g. http clients) to be used in tests that implement test timeouts.

Cancellation needs to be cooperative, similar to [kotlin coroutines](https://kotlinlang.org/docs/cancellation-and-timeouts.html#cancellation-is-cooperative).

Allow users to manually cancel tests.

### Test Labels

Attach labels to tests to easily filter for them later.

### Chart tests

Maybe we can use handoff in [helm chart tests](https://helm.sh/docs/topics/chart_tests/) to make sure that new helm chart deployments are working as expected.

### Github Integration

After a PR is merged and deployed we can update the PR with the results of the automated handoff test

### Fetch & Link Observability data

Prometheus / Loki / Tempo / ELK stack - find and fetch logs/traces/metrics that are created by tests (e.g. for easier debugging) - e.g. via correlation ids

### Soft test Fails

Tests that don't fail the entire testsuite. This can be used to help with the chicken/egg problem when you add new tests that target a new service version
that is not deployed yet.

### Configurable test retention

Delete old test runs after e.g. X days.

### Stream test results to UI or CLI

Create a websocket / SSE endpoint that can stream test results / logs.

### Grafana service dashboard template

Create a grafana dashboard that displays the exported prometheus metrics from handoff in useful graphs.

### Per SUT/Service view on tests

Service dashboards that show information of services k8s resources running in a cluster and their test suite runs..

### Support running tests in languages other than go

It should be possible to run external tests, e.g. java tests via the junit test runner CLI command and persist the results in handoff. It's unclear which features of handoff we can support in this mode.

### Display average test run duration

Calculate averages on how long individual test runs took in the past and display that in the UI, either as a duration string ("this test typically takes "4-6 seconds") or as a progress bar.

### K8s CRDs

Configure Test Schedules (and maybe other things) through k8s CRDs.

### Automatic test retries

Automatic test run retries/backoff on failures.

### Parallel test runs

Configurable parallel test runs within a test suite run.

### Query engine

Searchable test runs

### Aggregation Frontend

Separate server that serves a frontend that can aggregate results from multiple instances of handoff (dev, staging, production).
