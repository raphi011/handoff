package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
)

func TestMigration(t *testing.T) {
	s, err := storage.New("")
	if err != nil {
		t.Error(err)
	}
	defer close(s)

	ctx := context.Background()

	now := time.Now()

	id, err := s.SaveTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName:    "test",
		Result:       model.ResultPassed,
		TestFilter:   "",
		Tests:        5,
		Scheduled:    now,
		Start:        now,
		End:          time.Time{},
		DurationInMS: 0,
		SetupLogs:    "Log 1\nLog 2",
		TriggeredBy:  "web",
	})
	if err != nil {
		t.Error(err)
	}

	t.Logf("created testsuiterun with id %d", id)

	tsr, err := s.LoadTestSuiteRun(ctx, "test", id)
	if err != nil {
		t.Error(err)
	}

	err = s.UpdateTestSuiteRun(ctx, tsr)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%+v", tsr)
}

func TestUpsertTestRun(t *testing.T) {
	s, err := storage.New("")
	if err != nil {
		t.Error(err)
	}
	defer close(s)

	ctx := context.Background()

	now := time.Now()

	id, err := s.SaveTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName:    "test",
		Result:       model.ResultPassed,
		TestFilter:   "",
		Tests:        5,
		Scheduled:    now,
		Start:        now,
		End:          time.Time{},
		DurationInMS: 0,
		SetupLogs:    "Log 1",
		TriggeredBy:  "web",
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

func close(s *storage.Storage) {
	_ = s.Close()
}
