package main

import (
	"github.com/raphi011/handoff"
)

func TestSuites() []handoff.TestSuite {

	return []handoff.TestSuite{
		{
			Name: "plugin-test",
			Tests: []handoff.TestFunc{
				PluginTestSuccess,
			},
		},
	}
}

func TestSchedules() []handoff.ScheduledRun {
	return []handoff.ScheduledRun{{TestSuiteName: "plugin-test", Schedule: "@every 5s"}}
}

func PluginTestSuccess(t handoff.TB) {
	t.Log("Plugin test success")
}
