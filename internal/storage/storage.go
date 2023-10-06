package storage

import (
	"context"

	"github.com/raphi011/handoff/internal/model"
)

type Storage interface {
	InsertTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error
	UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error
	LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error)
	LoadPendingTestSuiteRuns(ctx context.Context) ([]model.TestSuiteRun, error)
	LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error)
	LoadTestRuns(ctx context.Context, suiteName string, tsrID int) ([]*model.TestRun, error)
	LoadTestRun(ctx context.Context, suiteName string, tsrID int, testName string) ([]model.TestRun, error)
	InsertTestRun(ctx context.Context, tr model.TestRun) error
	UpdateTestRun(ctx context.Context, tr model.TestRun) error
	StartTransaction(cx context.Context) (context.Context, error)
	CommitTransaction(cx context.Context) error
	RollbackTransaction(cx context.Context)
	Close() error
}
