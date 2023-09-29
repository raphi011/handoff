package handoff

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandoff(t *testing.T) {

	h := New(
		WithTestSuite(TestSuite{
			Name: "my-app",
			Tests: []TestFunc{
				Flaky,
			},
		}),
	)

	h.Run()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	h.httpServer.Handler.ServeHTTP(res, req)
}

func Flaky(t TB) {
	if rand.Intn(3) == 0 {
		t.Fatal("flaky test failed")
	}

	t.Log("flaky test succeeded")
}
