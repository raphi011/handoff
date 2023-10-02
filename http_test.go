package handoff_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/raphi011/handoff/client"
)

func TestSuiteRunWithUnknownSuiteShouldFailSuiteNotFoundReturns404(t *testing.T) {
	t.Parallel()

	i := acceptanceTest(t)
	defer i.shutdown()

	_, err := i.client.CreateTestSuiteRun(context.Background(), "not-found", nil)

	var reqError client.RequestError

	if !errors.As(err, &reqError) {
		t.Errorf("expected error of type RequestError but got %T: %v", err, err)
	}

	if reqError.ResponseCode != http.StatusNotFound {
		t.Errorf("expect response code %d but got %d", http.StatusNotFound, reqError.ResponseCode)
	}
}

func TestCreateTestSuiteRunShouldSucceed(t *testing.T) {
	t.Parallel()

	i := acceptanceTest(t)
	defer i.shutdown()

	_, err := i.client.CreateTestSuiteRun(context.Background(), "succeed", nil)
	if err != nil {
		t.Errorf("create test suite run should not fail: %v", err)
	}
}
