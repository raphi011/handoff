package storage

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/raphi011/handoff/internal/model"

	"golang.org/x/exp/slog"
)

//go:embed migrations/*.sql
var fs embed.FS

type Storage struct {
	db *sqlx.DB
}

func New(dbFilename string) (*Storage, error) {
	db, err := sqlx.Connect("sqlite", connectionString(dbFilename))
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	row := db.QueryRow("select sqlite_version()")

	var version string
	err = row.Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve sqlite version: %w", err)
	}

	slog.Info("Using sqlite version: " + version)

	if err = migrateDB(db); err != nil {
		return nil, err
	}

	return &Storage{
		db: db,
	}, nil
}

var pragma = []string{"busy_timeout(5000)", "journal_mode(WAL)", "foreign_keys(1)", "synchronous(normal)"}

func connectionString(filename string) string {
	var cs string

	if filename != "" {
		cs = filename
	} else {
		cs = ":memory:"
	}

	for i, p := range pragma {
		if i == 0 {
			cs += "?"
		} else {
			cs += "&"
		}

		cs += "_pragma=" + p
	}

	return cs
}

func migrateDB(db *sqlx.DB) error {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("load db migrations: %w", err)
	}

	driver, err := sqlite.WithInstance(db.DB, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("load migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate with instance: %w", err)
	}

	err = m.Up()

	if err == migrate.ErrNoChange {
		slog.Info("No migrations have been applied. The DB is at the latest state.")
	} else if err != nil {
		return fmt.Errorf("applying db migrations: %w", err)
	}

	return nil
}

func (s *Storage) SaveTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) (int, error) {
	logs := ""
	if len(tsr.SetupLogs) > 0 {
		logs = strings.Join(tsr.SetupLogs, "\n")
	}

	r, err := s.db.NamedExecContext(ctx, `INSERT INTO TestSuiteRun 
	(suiteName, result, testFilter, total, passed, skipped, failed, scheduledTime, startTime, endTime, setupLogs, triggeredBy) VALUES
	(:suiteName, :result, :testFilter, :total, :passed, :skipped, :failed, :scheduledTime, :startTime, :endTime, :setupLogs, :triggeredBy)`,
		map[string]any{
			"suiteName":     tsr.SuiteName,
			"result":        tsr.Result,
			"testFilter":    tsr.TestFilter,
			"total":         tsr.Tests,
			"passed":        tsr.Passed,
			"skipped":       tsr.Skipped,
			"failed":        tsr.Failed,
			"scheduledTime": timeFormat(tsr.Scheduled),
			"startTime":     timeFormat(tsr.Start),
			"endTime":       timeFormat(tsr.End),
			"setupLogs":     logs,
			"triggeredBy":   tsr.TriggeredBy,
		})
	if err != nil {
		return -1, err
	}

	id, err := r.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("retrieving inserted TestSuiteRun id: %w", err)
	}

	return int(id), nil
}

func (s *Storage) UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	logs := ""
	if len(tsr.SetupLogs) > 0 {
		logs = strings.Join(tsr.SetupLogs, "\n")
	}

	_, err := s.db.NamedExecContext(ctx, `UPDATE TestSuiteRun SET
	result=:result, passed=:passed, skipped=:skipped, failed=:failed, startTime=:startTime, endTime=:endTime, setupLogs=:setupLogs
	where id = :id and suiteName = :suiteName`,
		map[string]any{
			"result":    tsr.Result,
			"passed":    tsr.Passed,
			"skipped":   tsr.Skipped,
			"failed":    tsr.Failed,
			"startTime": timeFormat(tsr.Start),
			"endTime":   timeFormat(tsr.End),
			"setupLogs": logs,
			"id":        tsr.ID,
			"suiteName": tsr.SuiteName,
		})

	return err
}

func (s *Storage) LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error) {
	r, err := s.db.NamedQueryContext(ctx, `SELECT 
	id, suiteName, result, testFilter, total, passed, skipped, failed, scheduledTime, startTime, endTime, setupLogs, triggeredBy
	FROM TestSuiteRun WHERE suiteName = :suiteName and id = :id`,
		map[string]any{
			"suiteName": suiteName,
			"id":        runID,
		})
	if err != nil {
		return model.TestSuiteRun{}, err
	}
	defer r.Close()

	if !r.Next() {
		return model.TestSuiteRun{}, model.NotFoundError{}
	}

	return scanTestSuiteRun(r)
}

func (s *Storage) LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error) {
	runs := []model.TestSuiteRun{}
	r, err := s.db.NamedQueryContext(ctx, `SELECT 
		id, suiteName, result, testFilter, total, passed, skipped, failed, scheduledTime, startTime, endTime, setupLogs, triggeredBy
		FROM TestSuiteRun WHERE suiteName = :suiteName`,
		map[string]any{"suiteName": suiteName},
	)
	if err != nil {
		return runs, err
	}
	defer r.Close()

	for r.Next() {
		tsr, err := scanTestSuiteRun(r)
		if err != nil {
			return nil, err
		}

		runs = append(runs, tsr)
	}

	return runs, nil
}

func (s *Storage) UpsertTestRun(ctx context.Context, tsrID int, tr model.TestRun) error {
	logs := ""
	if len(tr.Logs) > 0 {
		logs = strings.Join(tr.Logs, "\n")
	}

	_, err := s.db.NamedExecContext(ctx, `INSERT INTO TestRun
	(suiteRunId, testName, passed, skipped, logs, startTime, endTime) VALUES
	(:suiteRunId, :testName, :passed, :skipped, :logs, :startTime, :endTime)
	ON CONFLICT(suiteRunID, testName) 
	DO UPDATE SET 
	passed=:passed, skipped=:skipped, logs=:logs, startTime=:startTime, endTime=:endTime`,
		map[string]any{
			"suiteRunId": tsrID,
			"testName":   tr.Name,
			"passed":     tr.Passed,
			"skipped":    tr.Skipped,
			"logs":       logs,
			"startTime":  timeFormat(tr.Start),
			"endTime":    timeFormat(tr.End),
		})

	return err
}

func timeFormat(t time.Time) string {
	return t.Format(time.RFC3339)
}

func parseDate(t string) (time.Time, error) {
	return time.Parse(time.RFC3339, t)
}

func scanTestSuiteRun(r *sqlx.Rows) (model.TestSuiteRun, error) {
	tsr := model.TestSuiteRun{}

	var start, end, scheduled string

	var logs string

	err := r.Scan(
		&tsr.ID,
		&tsr.SuiteName,
		&tsr.Result,
		&tsr.TestFilter,
		&tsr.Tests,
		&tsr.Passed,
		&tsr.Skipped,
		&tsr.Failed,
		&scheduled,
		&start,
		&end,
		&logs,
		&tsr.TriggeredBy,
	)
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("scanning test suite run: %w", err)
	}

	tsr.SetupLogs = strings.Split(logs, "\n")

	if tsr.Start, err = parseDate(start); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing start time: %w", err)
	}
	if tsr.End, err = parseDate(end); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing end time: %w", err)
	}
	if tsr.Scheduled, err = parseDate(scheduled); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing scheduled time: %w", err)
	}

	if tsr.TestFilter != "" {
		tsr.TestFilterRegex, err = regexp.Compile(tsr.TestFilter)
		if err != nil {
			return model.TestSuiteRun{}, fmt.Errorf("compiling test filter regex %s: %w", tsr.TestFilter, err)
		}
	}

	return tsr, nil
}
