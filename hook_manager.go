package handoff

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	// "github.com/raphi011/handoff/internal/hook"
	"github.com/raphi011/handoff/internal/model"
)

type TestFinishedListener interface {
	Hook
	TestFinished(suite model.TestSuite, run model.TestSuiteRun, testName string, context model.TestContext)
}

type AsyncTestFinishedListener interface {
	Hook
	TestFinishedAsync(suite model.TestSuite, run model.TestSuiteRun, testName string, context map[string]any, callback AsyncHookCallback)
}

type TestSuiteFinishedListener interface {
	Hook
	TestSuiteFinished(suite model.TestSuite, run model.TestSuiteRun)
}

type AsyncTestSuiteFinishedListener interface {
	Hook
	TestSuiteFinishedAsync(suite model.TestSuite, run model.TestSuiteRun, callback func(context map[string]any))
}

// AsyncHookCallback allows async hooks to add additional context
// to a testsuite or testrun.
type AsyncHookCallback func(context map[string]any)

type Hook interface {
	Name() string
	Init() error
}

type hookManager struct {
	all                    []Hook
	testFinished           []TestFinishedListener
	testFinishedAsync      []AsyncTestFinishedListener
	testSuiteFinished      []TestSuiteFinishedListener
	testSuiteFinishedAsync []AsyncTestSuiteFinishedListener

	asyncCallback asyncHookCallback

	asyncHooksRunning sync.WaitGroup

	log *slog.Logger
}

type asyncHookCallback func(p Hook, context map[string]any)

func newHookManager(hookCallback asyncHookCallback, log *slog.Logger) *hookManager {
	return &hookManager{
		all:                    []Hook{},
		testFinished:           []TestFinishedListener{},
		testFinishedAsync:      []AsyncTestFinishedListener{},
		testSuiteFinished:      []TestSuiteFinishedListener{},
		testSuiteFinishedAsync: []AsyncTestSuiteFinishedListener{},

		asyncCallback: hookCallback,
		log:           log,
	}
}

func (s *hookManager) init() error {
	// for testing purposes
	// s.all = append(s.all, hook.NewSlackHook("", "", s.log))

	for _, p := range s.all {
		if err := p.Init(); err != nil {
			return fmt.Errorf("initiating hook %q: %w", p.Name(), err)
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
		if l, ok := p.(AsyncTestSuiteFinishedListener); ok {
			s.testSuiteFinishedAsync = append(s.testSuiteFinishedAsync, l)
			registeredHook = true
		}

		if !registeredHook {
			return fmt.Errorf("hook %q does not implement any listener", p.Name())
		}
	}

	return nil
}

func (s *hookManager) shutdown() context.Context {
	cancelCtx, cancel := context.WithCancel(context.Background())

	go func() {
		s.asyncHooksRunning.Wait()
		cancel()
	}()

	return cancelCtx
}

func (s *hookManager) notifyTestSuiteFinished(suite model.TestSuite, testSuiteRun model.TestSuiteRun) {
	for _, p := range s.testSuiteFinished {
		p.TestSuiteFinished(suite, testSuiteRun)
	}
}

func (s *hookManager) notifyTestSuiteFinishedAsync(suite model.TestSuite, testSuiteRun model.TestSuiteRun) {
	for _, p := range s.testSuiteFinishedAsync {
		s.asyncHooksRunning.Add(1)

		hook := p
		go func() {
			// TODO catch panics
			defer s.asyncHooksRunning.Done()
			hook.TestSuiteFinishedAsync(suite, testSuiteRun, s.newAsyncHookCallback(hook))
		}()
	}
}

func (s *hookManager) notifyTestFinished(suite model.TestSuite, testRun model.TestSuiteRun, name string, runContext model.TestContext) {
	for _, p := range s.testFinished {
		p.TestFinished(suite, testRun, name, runContext)
	}
}

func (s *hookManager) notifyTestFinishedAync(suite model.TestSuite, testRun model.TestSuiteRun, name string, runContext map[string]any) {
	for _, p := range s.testFinishedAsync {
		s.asyncHooksRunning.Add(1)

		hook := p
		go func() {
			// TODO catch panics
			defer s.asyncHooksRunning.Done()
			hook.TestFinishedAsync(suite, testRun, name, runContext, s.newAsyncHookCallback(hook))
		}()
	}
}

func (s *hookManager) newAsyncHookCallback(p Hook) AsyncHookCallback {
	return func(c map[string]any) {
		s.asyncCallback(p, c)
	}
}
