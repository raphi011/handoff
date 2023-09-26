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

	ctx := context.Background()

	now := time.Now()

	id, err := s.SaveTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName:    "test",
		Result:       model.ResultPassed,
		TestFilter:   "",
		Tests:        5,
		Passed:       3,
		Skipped:      1,
		Failed:       1,
		Scheduled:    now,
		Start:        now,
		End:          time.Time{},
		DurationInMS: 0,
		SetupLogs:    []string{"Log 1", "Log 2"},
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
	s, err := storage.New("ab.db")
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	now := time.Now()

	id, err := s.SaveTestSuiteRun(ctx, model.TestSuiteRun{
		SuiteName:    "test",
		Result:       model.ResultPassed,
		TestFilter:   "",
		Tests:        5,
		Passed:       3,
		Skipped:      1,
		Failed:       1,
		Scheduled:    now,
		Start:        now,
		End:          time.Time{},
		DurationInMS: 0,
		SetupLogs:    []string{"Log 1", "Log 2"},
		TriggeredBy:  "web",
	})
	if err != nil {
		t.Error(err)
	}

	t.Logf("created testsuiterun with id %d", id)

	tr := model.TestRun{
		Name: "bla",
	}

	err = s.UpsertTestRun(ctx, id, tr)
	if err != nil {
		t.Error(err)
	}

	tr.Passed = true

	err = s.UpsertTestRun(ctx, id, tr)
	if err != nil {
		t.Error(err)
	}
}
