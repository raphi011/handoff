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

func (c *TsrCache) Save(tsr model.TestSuiteRun) {
	c.m.Store(testSuiteRunKey(tsr.SuiteName, tsr.ID), tsr)
}

func (c *TsrCache) Load(suiteName string, id int) (model.TestSuiteRun, error) {
	val, ok := c.m.Load(testSuiteRunKey(suiteName, id))
	if !ok {
		return model.TestSuiteRun{}, model.NotFoundError{}
	}

	return val.(model.TestSuiteRun), nil
}

func (c *TsrCache) LoadAndDelete(suiteName string, id int) (model.TestSuiteRun, error) {
	val, ok := c.m.LoadAndDelete(testSuiteRunKey(suiteName, id))
	if !ok {
		return model.TestSuiteRun{}, model.NotFoundError{}
	}

	return val.(model.TestSuiteRun), nil
}
