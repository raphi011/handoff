package main

import (
	"github.com/raphi011/handoff"
)

func Handoff() ([]handoff.TestSuite, []handoff.ScheduledRun) {

	return []handoff.TestSuite{
			{
				Name: "plugin-test",
				Tests: []handoff.TestFunc{
					PluginTestSuccess,
				},
			},
			{
				Name: "plugin-scheduled-test",
				Tests: []handoff.TestFunc{
					PluginTestSuccess,
				},
			},
		}, []handoff.ScheduledRun{{
			TestSuiteName: "plugin-scheduled-test", Schedule: "@every 2s"},
		}
}

func PluginTestSuccess(t handoff.TB) {
	t.Log("Plugin test success")
}
