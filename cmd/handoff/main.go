package main

import (
	"os"

	"github.com/raphi011/handoff"
	"golang.org/x/exp/slog"
)

func main() {
	h := handoff.New()

	if err := h.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(-1)
	}
}
