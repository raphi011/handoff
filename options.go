package handoff

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/raphi011/handoff/internal/model"
)

// WithServerPort sets the port that is used by the server.
// This can be overridden by flags.
func WithServerPort(port int) option {
	return func(s *Handoff) {
		s.port = port
	}
}

// WithScheduledRun schedules a TestSuite to run at certain intervals.
// Ignored in CLI mode.
func WithScheduledRun(sr ScheduledRun) option {
	return func(s *Handoff) {
		s.schedules = append(s.schedules, sr)
	}
}

func WithPlugin(p Plugin) option {
	return func(s *Handoff) {
		s.plugins.all = append(s.plugins.all, p)
	}
}

func WithTestSuite(suite TestSuite) option {
	return func(s *Handoff) {
		s.readOnlyTestSuites[suite.Name] = mapTestSuite(suite)
	}
}

func mapTestSuite(ts TestSuite) model.TestSuite {
	mapped := model.TestSuite{
		Name:              ts.Name,
		AssociatedService: ts.AssociatedService,
		Setup:             ts.Setup,
		Teardown:          ts.Teardown,
		Tests:             make(map[string]model.TestFunc),
	}
	for _, t := range ts.Tests {
		mapped.Tests[testName(t)] = t
	}

	return mapped
}

func testName(tf TestFunc) string {
	fullFuncName := runtime.FuncForPC(reflect.ValueOf(tf).Pointer()).Name()

	packageIndex := strings.LastIndex(fullFuncName, ".") + 1
	// remove the package name
	return fullFuncName[packageIndex:]
}
