package storage

import (
	"sync"

	"github.com/raphi011/handoff/internal/model"
)

type TsrCache struct {
	m sync.Map
}

func NewTsrCache() *TsrCache {
	return &TsrCache{}
}

func (c *TsrCache) Save(tsr *model.TestSuiteRun) {
	c.m.Store(string(testSuiteRunKey(tsr.SuiteName, tsr.ID)), tsr)
}

func (c *TsrCache) Load(suiteName string, id int) (model.TestSuiteRun, bool) {
	val, ok := c.m.Load(string(testSuiteRunKey(suiteName, id)))
	if !ok {
		return model.TestSuiteRun{}, false
	}

	return *val.(*model.TestSuiteRun), true

	// tsrCopy := *tsr

	// copy(tsrCopy.TestResults, tsr.TestResults)

	// return b, true
}

func (c *TsrCache) LoadAndDelete(suiteName string, id int) (model.TestSuiteRun, error) {
	val, ok := c.m.LoadAndDelete(string(testSuiteRunKey(suiteName, id)))
	if !ok {
		return model.TestSuiteRun{}, model.NotFoundError{}
	}

	return val.(model.TestSuiteRun), nil
}
