package handoff

import (
	"context"
	"fmt"
	"time"

	"github.com/raphi011/handoff/internal/metric"
	"github.com/raphi011/handoff/internal/model"
)

// runTestSuite executes a test suite run. It will run all tests that are either pending
// or can be retried (attempt<maxattempts).
func (s *Server) runTestSuite(
	suite model.TestSuite,
	tsr model.TestSuiteRun,
) {
	s.runningTestSuites.Add(1)
	defer s.runningTestSuites.Done()

	if tsr.Result == model.ResultPassed {
		// nothing to do here
		return
	}

	ctx := context.Background()

	log := s.log.With("suite-name", suite.Name, "run-id", tsr.ID)

	tsr.Start = time.Now()

	testSuitesRunning := metric.TestSuitesRunning.WithLabelValues(s.config.Instance, suite.Namespace, suite.Name)
	testSuitesRunning.Inc()
	defer func() {
		testSuitesRunning.Dec()
	}()

	if err := suite.SafeSetup(); err != nil {
		log.Warn("setup of suite failed", "error", err)
		end := time.Now()

		tsr.Result = model.ResultFailed
		tsr.SetupLogs = fmt.Sprintf("setup failed: %v", err)

		for i := 0; i < len(tsr.TestResults); i++ {
			tr := &tsr.TestResults[i]

			if tr.Result == model.ResultPending {
				tr.Result = model.ResultSkipped
				tr.Logs = "test suite run setup failed: skipped"
				tr.End = end
			}
		}
	}

	// skip if setup failed
	if tsr.Result != model.ResultFailed {
		for i := 0; i < len(tsr.TestResults); i++ {
			tr := &tsr.TestResults[i]

			if s.isShuttingDown() {
				break
			}

			if tr.Result != model.ResultPending {
				continue
			}

			s.runTest(ctx, suite, tsr, tr)

			if tsr.ShouldRetry(*tr) {
				newAttempt := tr.NewAttempt()

				tsr.TestResults = append(tsr.TestResults, newAttempt)
			}
		}

		if err := suite.SafeTeardown(); err != nil {
			log.Warn("teardown of suite failed", "error", err)
		}

		tsr.Result = tsr.ResultFromTestResults()
	}

	if tsr.Result != model.ResultPending {
		tsr.End = time.Now()
	}
	tsr.DurationInMS = tsr.TestSuiteDuration()
	tsr.Flaky = tsr.IsFlaky()

	s.hooks.notifyTestSuiteFinished(suite, tsr)

	if err := s.storage.UpdateTestSuiteRun(ctx, tsr); err != nil {
		log.Error("updating test suite run failed", "error", err)
	}

	if tsr.Result != model.ResultPending {
		s.hooks.notifyTestSuiteFinishedAsync(suite, tsr)

		metric.TestSuiteFinished(s.config.Instance, suite, tsr)
	}
}

// runTest runs an individual test that is part of a test suite. This function must only be called
// by `runTestSuite()`.
func (s *Server) runTest(
	_ context.Context,
	suite model.TestSuite,
	testSuiteRun model.TestSuiteRun,
	testRun *model.TestRun,
) {
	t := T{
		attempt:        testRun.Attempt,
		suiteName:      suite.Name,
		testName:       testRun.Name,
		runtimeContext: map[string]any{},
	}

	start := time.Now()

	defer func() {
		end := time.Now()

		err := recover()

		result := t.Result()

		metric.TestRunsTotal.WithLabelValues(s.config.Instance, suite.Namespace, suite.Name, string(result)).Inc()

		s.hooks.notifyTestFinished(suite, testSuiteRun, testRun.Name, t.runtimeContext)

		logs := t.logs

		if err != nil && t.result != model.ResultSkipped {
			if _, ok := err.(failTestErr); !ok {
				// this is an unexpected panic (does not originate from handoff)
				logs.WriteString(fmt.Sprintf("%v\n", err))
				result = model.ResultFailed
			}
		}

		testRun.Start = start
		testRun.End = end
		testRun.DurationInMS = end.Sub(start).Milliseconds()
		testRun.Result = result
		testRun.SoftFailure = t.softFailure
		testRun.Logs = logs.String()
		testRun.Context = t.runtimeContext
		testRun.Spans = t.spans

		s.hooks.notifyTestFinishedAync(suite, testSuiteRun, testRun.Name, t.runtimeContext)

		if err = t.runTestCleanup(); err != nil {
			s.log.Warn("test cleanup failed", "suite-name", suite.Name, "error", err)
		}
	}()

	suite.Tests[testRun.Name](&t)
}
