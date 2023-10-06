package handoff_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/raphi011/handoff"
	"github.com/raphi011/handoff/client"
	"github.com/raphi011/handoff/internal/model"
	"github.com/stretchr/testify/assert"

	_ "github.com/raphi011/handoff/internal/packagetestexample"
)

var te *instance

const (
	defaultTimeout = 3 * time.Second
)

func TestMain(m *testing.M) {
	args := os.Args

	te = handoffInstance(defaultTestSuites, nil, []string{"handoff-test", "-p", "0", "-d", ""})

	// restore test command args, as m.Run() requires them.
	os.Args = args

	code := m.Run()

	te.h.Shutdown()

	os.Exit(code)
}

func Fail(t handoff.TB) {
	t.Fail()
}

func Flaky(t handoff.TB) {
	if rand.Intn(3) == 0 {
		t.Fatal("flaky test failed")
	}

	t.Log("flaky test succeeded")
}

func SoftFail(t handoff.TB) {
	t.SoftFailure()

	t.Error("Soft fail error")
}

func Retry(x int) handoff.TestFunc {
	return func(t handoff.TB) {
		if t.Attempt() < x+1 {
			t.Fatalf("This tests fails before the %d attempt", x)
		}
	}
}

func Sleep(sleep time.Duration) handoff.TestFunc {
	return func(t handoff.TB) {
		time.Sleep(sleep)
	}
}

func Success(t handoff.TB) {
	t.Log("Success")
}

func LogAttempt(t handoff.TB) {
	t.Logf("Attempt: %d", t.Attempt())
}

type instance struct {
	h      *handoff.Server
	client client.Client
}

var defaultTestSuites = []handoff.TestSuite{
	{
		Name: "succeed",
		Tests: []handoff.TestFunc{
			Success,
		},
	},
	{
		Name: "panicing-setup",
		Setup: func() error {
			panic("panic")
		},
		Tests: []handoff.TestFunc{
			Success,
		},
	},
	{
		Name:            "multiple-tests",
		MaxTestAttempts: 2,
		Tests: []handoff.TestFunc{
			Success,
			Flaky,
			Retry(1),
		},
	},
	{
		Name: "failing-setup",
		Setup: func() error {
			return errors.New("error")
		},
		Tests: []handoff.TestFunc{
			Success,
		},
	},
	{
		Name:            "needs-retry",
		MaxTestAttempts: 2,
		Tests: []handoff.TestFunc{
			Retry(1),
		},
	},
	{
		Name: "soft-fail",
		Tests: []handoff.TestFunc{
			Success,
			SoftFail,
		},
	},
	{
		Name: "my-app",
		Tests: []handoff.TestFunc{
			Flaky,
		},
	},
	{
		Name: "failing",
		Tests: []handoff.TestFunc{
			Fail,
		},
	},
}

func handoffInstance(
	suites []handoff.TestSuite,
	schedules []handoff.ScheduledRun,
	args []string,
) *instance {
	os.Args = args

	var options []handoff.Option

	for _, ts := range suites {
		options = append(options, handoff.WithTestSuite(ts))
	}

	for _, s := range schedules {
		options = append(options, handoff.WithScheduledRun(s))
	}

	h := handoff.New(options...)

	go h.Run()

	h.WaitForStartup()

	port := h.ServerPort()

	return &instance{
		h:      h,
		client: client.New(fmt.Sprintf("http://localhost:%d", port), http.DefaultClient),
	}
}

func latestTestAttempt(t *testing.T, tsr client.TestSuiteRun, testName string) client.TestRun {
	t.Helper()

	var tr client.TestRun

	for _, r := range tsr.TestResults {
		if r.Name == testName && r.Attempt > tr.Attempt {
			tr = r
		}
	}

	assert.NotEmptyf(t, tr.Name, "could not find any attempt for test name %s", testName)

	return tr
}

func (ti *instance) createNewTestSuiteRun(t *testing.T, suiteName string) client.TestSuiteRun {
	tsr, err := ti.client.CreateTestSuiteRun(context.Background(), suiteName, nil)
	assert.NoError(t, err, "unable to create test suite run")

	return tsr
}

func (ti *instance) waitForTestSuiteRunWithResult(t *testing.T, timeout time.Duration, suiteName string, runID int, status model.Result) client.TestSuiteRun {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		tsr, err := ti.client.GetTestSuiteRun(ctx, suiteName, runID)
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timed out waiting for test suite run with status %s", status)
			return model.TestSuiteRunHTTP{}
		} else if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		if tsr.Result == status {
			return tsr
		} else if tsr.Result != model.ResultPending {
			t.Fatalf("test suite run result is %q, expected %q", tsr.Result, status)
		}
	}
}
