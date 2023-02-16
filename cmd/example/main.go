package main

import (
	"log"
	"time"

	"github.com/raphi011/handoff"
	"github.com/stretchr/testify/assert"
)

func main() {
	s := handoff.New(
		handoff.WithTestSuite(handoff.TestSuite{
			Name: "my-app",
			Tests: map[string]handoff.TestFunc{
				"TestSleep":   TestSleep,
				"TestSuccess": TestSuccess,
				"TestPanic":   TestPanic,
				"TestSkip":    TestSkip,
				"TestFatal":   TestFatal,
				"TestTestify": TestTestify,
			},
		}),
		handoff.WithScheduledRun("my-app", "@every 5s"),
	)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

func TestSleep(t handoff.TB) {
	time.Sleep(1 * time.Second)
}

func TestSuccess(t handoff.TB) {
	t.Log("Executed TestAcceptance")
}

func TestFatal(t handoff.TB) {
	t.Fatal("fatal error")
}

func TestPanic(t handoff.TB) {
	panic("panic!")
}

func TestSkip(t handoff.TB) {
	t.Skip("skipping test")
}

func TestTestify(t handoff.TB) {
	assert.Equal(t, 1, 2)
}
