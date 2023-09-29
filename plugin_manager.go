package handoff

import (
	"context"
	"fmt"
	"sync"

	"github.com/raphi011/handoff/internal/model"
)

type TestFinishedListener interface {
	Plugin
	TestFinished(suite model.TestSuite, run model.TestSuiteRun, testName string, context model.TestContext)
}

type AsyncTestFinishedListener interface {
	Plugin
	TestFinishedAsync(suite model.TestSuite, run model.TestSuiteRun, testName string, context map[string]any, callback AsyncPluginCallback)
}

type TestSuiteFinishedListener interface {
	Plugin
	TestSuiteFinished(suite model.TestSuite, run model.TestSuiteRun)
}

type AsyncTestSuiteFinishedListener interface {
	Plugin
	TestSuiteFinishedAsync(suite model.TestSuite, run model.TestSuiteRun, callback AsyncPluginCallback)
}

// AsyncPluginCallback allows async plugin handlers to add additional context
// to a testsuite or testrun.
type AsyncPluginCallback func(context map[string]any)

type Plugin interface {
	Name() string
	Init() error
}

type pluginManager struct {
	all                    []Plugin
	testFinished           []TestFinishedListener
	testFinishedAsync      []AsyncTestFinishedListener
	testSuiteFinished      []TestSuiteFinishedListener
	testSuiteFinishedAsync []AsyncTestSuiteFinishedListener

	asyncCallback asyncPluginCallback

	asyncHooksRunning sync.WaitGroup
}

type asyncPluginCallback func(p Plugin, context map[string]any)

func newPluginManager(pluginCallback asyncPluginCallback) *pluginManager {
	return &pluginManager{
		all:                    []Plugin{},
		testFinished:           []TestFinishedListener{},
		testFinishedAsync:      []AsyncTestFinishedListener{},
		testSuiteFinished:      []TestSuiteFinishedListener{},
		testSuiteFinishedAsync: []AsyncTestSuiteFinishedListener{},

		asyncCallback: pluginCallback,
	}
}

func (s *pluginManager) init() error {
	for _, p := range s.all {
		if err := p.Init(); err != nil {
			return fmt.Errorf("initiating plugin %q: %w", p.Name(), err)
		}

		registeredHook := false

		if l, ok := p.(TestFinishedListener); ok {
			s.testFinished = append(s.testFinished, l)
			registeredHook = true
		}
		if l, ok := p.(TestSuiteFinishedListener); ok {
			s.testSuiteFinished = append(s.testSuiteFinished, l)
			registeredHook = true
		}

		if !registeredHook {
			return fmt.Errorf("plugin %q does not implement any hook", p.Name())
		}
	}

	return nil
}

func (s *pluginManager) shutdown() context.Context {
	cancelCtx, cancel := context.WithCancel(context.Background())

	go func() {
		s.asyncHooksRunning.Wait()
		cancel()
	}()

	return cancelCtx
}

func (s *pluginManager) notifyTestSuiteFinished(suite model.TestSuite, testSuiteRun model.TestSuiteRun) {
	for _, p := range s.testSuiteFinished {
		p.TestSuiteFinished(suite, testSuiteRun)
	}
}

func (s *pluginManager) notifyTestSuiteFinishedAsync(suite model.TestSuite, testSuiteRun model.TestSuiteRun) {
	for _, p := range s.testSuiteFinishedAsync {
		s.asyncHooksRunning.Add(1)

		hook := p
		go func() {
			// TODO catch panics
			defer s.asyncHooksRunning.Done()
			hook.TestSuiteFinishedAsync(suite, testSuiteRun, s.newAsyncPluginCallback(hook))
		}()
	}
}

func (s *pluginManager) notifyTestFinished(suite model.TestSuite, testRun model.TestSuiteRun, name string, runContext model.TestContext) {
	for _, p := range s.testFinished {
		p.TestFinished(suite, testRun, name, runContext)
	}
}

func (s *pluginManager) notifyTestFinishedAync(suite model.TestSuite, testRun model.TestSuiteRun, name string, runContext map[string]any) {
	for _, p := range s.testFinishedAsync {
		s.asyncHooksRunning.Add(1)

		hook := p
		go func() {
			// TODO catch panics
			defer s.asyncHooksRunning.Done()
			hook.TestFinishedAsync(suite, testRun, name, runContext, s.newAsyncPluginCallback(hook))
		}()
	}
}

func (s *pluginManager) newAsyncPluginCallback(p Plugin) AsyncPluginCallback {
	return func(c map[string]any) {
		s.asyncCallback(p, c)
	}
}
