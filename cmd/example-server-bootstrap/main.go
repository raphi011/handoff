package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/raphi011/handoff"
	"github.com/raphi011/handoff/plugin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func main() {
	h := handoff.New(
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "soft-failure",
			Tests: []handoff.TestFunc{
				SoftFailure,
			},
		}),
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "my-app",
			Tests: []handoff.TestFunc{
				Flaky,
				Sleep,
				Success,
				Panic,
				Skip,
				Fatal,
				Testify,
			},
		}),
		handoff.WithScheduledRun(handoff.ScheduledRun{TestSuiteName: "my-app", Schedule: "@every 5s"}),
		handoff.WithPlugin(&plugin.ElasticSearchPlugin{}),
		handoff.WithServerPort(1337),
	)

	if err := h.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(-1)
	}
}

func SoftFailure(t handoff.TB) {
	t.SoftFailure()

	t.Fail()
}

func Flaky(t handoff.TB) {
	if rand.Intn(3) == 0 {
		t.Fatal("flaky test failed")
	}

	t.Log("flaky test succeeded")
}

func Sleep(t handoff.TB) {
	time.Sleep(1 * time.Second)
}

func Success(t handoff.TB) {
	t.Log("Executed TestAcceptance")
}

func Fatal(t handoff.TB) {
	t.Fatal("fatal error")
}

func Panic(t handoff.TB) {
	panic("panic!")
}

func Skip(t handoff.TB) {
	t.Skip("skipping test")
}

func Testify(t handoff.TB) {
	assert.Equal(t, 1, 2)
}
