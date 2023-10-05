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
)

var te *test

const (
	defaultTimeout = 3 * time.Second
)

func TestMain(m *testing.M) {
	te = acceptanceTest()

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

var hasRunBefore bool

func RetryOnce(t handoff.TB) {
	if !hasRunBefore {
		hasRunBefore = true
		t.Fatal("This tests fails the first time and then succeeds")
	}
}

func Success(t handoff.TB) {
	t.Log("Success")
}

type test struct {
	h      *handoff.Server
	client client.Client
}

func acceptanceTest() *test {
	// save go test args
	args := os.Args
	// random port and in-memory database
	os.Args = []string{"handoff-test", "-p", "0", "-d", ""}

	h := handoff.New(
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "succeed",
			Tests: []handoff.TestFunc{
				Success,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "panicing-setup",
			Setup: func() error {
				panic("panic")
			},
			Tests: []handoff.TestFunc{
				Success,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "failing-setup",
			Setup: func() error {
				return errors.New("error")
			},
			Tests: []handoff.TestFunc{
				Success,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name:            "needs-retry",
			MaxTestAttempts: 2,
			Tests: []handoff.TestFunc{
				RetryOnce,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "soft-fail",
			Tests: []handoff.TestFunc{
				Success,
				SoftFail,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "my-app",
			Tests: []handoff.TestFunc{
				Flaky,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "failing",
			Tests: []handoff.TestFunc{
				Fail,
			},
		}),
	)

	go h.Run()

	h.WaitForStartup()

	port := h.ServerPort()

	// restore go test args
	os.Args = args

	return &test{
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

	if tr.Name == "" {
		t.Fatalf("could not find any attempt for test name %s", testName)
	}

	return tr
}

func (ti *test) createNewTestSuiteRun(t *testing.T, suiteName string) client.TestSuiteRun {
	tsr, err := ti.client.CreateTestSuiteRun(context.Background(), suiteName, nil)
	if err != nil {
		t.Errorf("unable to create test suite run: %v", err)
	}

	return tsr
}

func (ti *test) waitForTestSuiteRunWithResult(t *testing.T, timeout time.Duration, suiteName string, runID int, status model.Result) client.TestSuiteRun {
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
