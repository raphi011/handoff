package handoff

import (
	"regexp"

	"github.com/robfig/cron/v3"
)

type ScheduledRun struct {
	// TestSuiteName is the name of the test suite to be run.
	TestSuiteName string
	// Schedule defines how often a run is scheduled. For the format see
	// https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format
	Schedule string
	// TestFilter allows enabling/filtering only certain tests of a testsuite to be run
	TestFilter *regexp.Regexp

	// EntryID identifies the cronjob
	EntryID cron.EntryID
}
