package main

import (
	"log/slog"
	"os"

	"github.com/raphi011/handoff"
)

func main() {
	h := handoff.New()

	if err := h.Run(); err != nil {
		slog.Error(err.Error())
		os.Exit(-1)
	}
}
