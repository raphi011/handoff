package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphi011/handoff/internal/model"
)

var (
	TestSuitesRunning = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "handoff_testsuites_running",
		Help: "The number of test suites currently running",
	}, []string{"instance", "namespace", "suite_name"})

	TestSuitesRun = promauto.NewCounterVec(prometheus.CounterOpts{Name: "handoff_testsuites_started_total",
		Help: "The number of test suite runs",
	}, []string{"instance", "namespace", "suite_name", "result", "flaky"})

	TestRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "handoff_tests_run_total",
		Help: "The number of tests run",
	}, []string{"instance", "namespace", "suite_name", "result"})
)

func TestSuiteFinished(instance string, suite model.TestSuite, tsr model.TestSuiteRun) {
	flaky := "0"
	if tsr.Flaky {
		flaky = "1"
	}

	TestSuitesRun.WithLabelValues(instance, suite.Namespace, suite.Name, string(tsr.Result), flaky).Inc()
}
