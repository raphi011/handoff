package main

import (
	"log"

	"github.com/raphi011/handoff"
)

func main() {
	s := handoff.New(handoff.WithTestSuite(handoff.TestSuite{
		Name: "my-app",
		Tests: map[string]handoff.TestFunc{
			"TestSuccess": TestSuccess,
			"TestPanic":   TestPanic,
		},
	}))

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

func TestSuccess(t handoff.TB) {
	t.Log("Executed TestAcceptance")
}

func TestPanic(t handoff.TB) {
	panic("deliberately fail")
}
