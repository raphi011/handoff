package storage_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestBadgerdb(t *testing.T) {
	db, err := storage.NewBadgerStorage("", slog.Default())
	assert.NoError(t, err)

	ctx, err := db.StartTransaction(context.Background())
	assert.NoError(t, err)

	suiteName := "test-suite"

	tr := model.TestRun{
		SuiteName: suiteName,
		Attempt:   1,
		Name:      "Bla",
	}

	id, err := db.InsertTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName:   suiteName,
		TestResults: []model.TestRun{tr},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, id)

	id, err = db.InsertTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName: suiteName,
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, id)

	_, err = db.LoadTestSuiteRun(ctx, suiteName, 1)
	assert.NoError(t, err)

	_, err = db.LoadTestSuiteRunsByName(ctx, suiteName)
	assert.NoError(t, err)
}
