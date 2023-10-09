package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TestSuitesRunning = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "handoff_testsuites_running",
		Help: "The number of test suites currently running",
	}, []string{"instance", "namespace", "suite_name"})

	TestSuitesRun = promauto.NewCounterVec(prometheus.CounterOpts{Name: "handoff_testsuites_started_total",
		Help: "The number of test suite runs",
	}, []string{"instance", "namespace", "suite_name", "result"})

	TestRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "handoff_tests_run_total",
		Help: "The number of tests run",
	}, []string{"instance", "namespace", "suite_name", "result"})
)
