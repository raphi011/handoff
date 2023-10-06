package storage_test

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	s, err := storage.NewSqlite(":memory:", slog.Default())
	if err != nil {
		t.Error(err)
	}
	defer close(s)

	ctx := context.Background()

	now := time.Now()
	tsr := model.TestSuiteRun{
		ID:        1,
		SuiteName: "test",
		Result:    model.ResultPassed,
		Params: model.RunParams{
			TestFilter:  regexp.MustCompile("Success|Skip"),
			TriggeredBy: "web",
		},
		Tests:        5,
		Scheduled:    now,
		Start:        now,
		End:          time.Time{},
		DurationInMS: 0,
		SetupLogs:    "Log 1\nLog 2",
	}

	err = s.InsertTestSuiteRun(ctx, tsr)
	if err != nil {
		t.Error(err)
	}

	tsr, err = s.LoadTestSuiteRun(ctx, "test", tsr.ID)
	if err != nil {
		t.Error(err)
	}

	err = s.UpdateTestSuiteRun(ctx, tsr)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%+v", tsr)
}

func TestLogCompression(t *testing.T) {
	s, err := storage.NewSqlite(":memory:", slog.Default())
	assert.NoError(t, err, "creating new storage instance should succeed")
	defer close(s)

	ctx := context.Background()

	suiteName := "test"
	testName := "Success"
	setupLogs := "setup log"
	testRunLogs := "test run log"

	err = s.InsertTestSuiteRun(ctx, model.TestSuiteRun{
		ID:        1,
		SuiteName: suiteName,
		SetupLogs: setupLogs,
	})
	assert.NoError(t, err, "inserting test suite run should succeed")

	tsr, err := s.LoadTestSuiteRun(ctx, suiteName, 1)
	assert.NoError(t, err, "loading test suite run should succeed")
	assert.Equal(t, setupLogs, tsr.SetupLogs)

	err = s.InsertTestRun(ctx, model.TestRun{
		SuiteName:  suiteName,
		Name:       testName,
		SuiteRunID: tsr.ID,
		Result:     model.ResultPassed,
		Attempt:    1,
		Logs:       testRunLogs,
	})
	assert.NoError(t, err, "inserting test run should succeed")

	tr, err := s.LoadTestRun(ctx, suiteName, tsr.ID, testName)
	assert.NoError(t, err, "loading test run should succeed")
	assert.Len(t, tr, 1)
	assert.Equal(t, testRunLogs, tr[0].Logs)
}

func TestUpdateTestRun(t *testing.T) {
	s, err := storage.NewSqlite(":memory:", slog.Default())
	assert.NoError(t, err, "creating new storage instance should succeed")
	defer close(s)

	ctx := context.Background()

	id := 1

	err = s.InsertTestSuiteRun(ctx, model.TestSuiteRun{
		ID:        id,
		SuiteName: "test",
		Result:    model.ResultPassed,
	})
	if err != nil {
		t.Error(err)
	}

	t.Logf("created testsuiterun with id %d", id)

	tr := model.TestRun{
		Name:       "bla",
		SuiteName:  "test",
		SuiteRunID: id,
		Attempt:    1,
		Result:     model.ResultPending,
	}

	err = s.InsertTestRun(ctx, tr)
	if err != nil {
		t.Error(err)
	}

	tr.Result = model.ResultPassed

	err = s.UpdateTestRun(ctx, tr)
	if err != nil {
		t.Error(err)
	}
}

func close(s storage.Storage) {
	_ = s.Close()
}
