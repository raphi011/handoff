package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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

func (s *Storage) Close() error {
	return s.db.Close()
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

type storageContextKey string

func (s *Storage) StartTransaction(ctx context.Context) (context.Context, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, storageContextKey("storage.transaction"), tx), nil
}

func (s *Storage) CommitTransaction(ctx context.Context) error {
	v := ctx.Value(storageContextKey("storage.transaction"))

	if v == nil {
		return errors.New("context does not contain a transaction")
	}

	return v.(*sqlx.Tx).Commit()
}

func (s *Storage) RollbackTransaction(ctx context.Context) {
	v := ctx.Value(storageContextKey("storage.transaction"))

	if v != nil {
		err := v.(*sqlx.Tx).Rollback()
		if err != nil && err != sql.ErrTxDone {
			slog.Warn("could not rollback transaction", "error", err)
		}
	}
}

func (s *Storage) getDB(ctx context.Context) commonDB {
	v := ctx.Value(storageContextKey("storage.transaction"))

	if v == nil {
		return s.db
	}

	return v.(*sqlx.Tx)
}

type commonDB interface {
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
}

func (s *Storage) SaveTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) (int, error) {
	db := s.getDB(ctx)

	r, err := db.NamedQuery(`INSERT INTO TestSuiteRun 
	(suiteName, environment, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, id) VALUES
	(:suiteName, :environment, :result, :testFilter, :scheduledTime, :startTime, :endTime, :setupLogs, :triggeredBy, 
		COALESCE((select max(id)+1 from TestSuiteRun where suiteName=:suiteName), 1))
	RETURNING id`,
		map[string]any{
			"suiteName":     tsr.SuiteName,
			"environment":   tsr.Environment,
			"result":        tsr.Result,
			"testFilter":    tsr.TestFilter,
			"scheduledTime": timeFormat(tsr.Scheduled),
			"startTime":     timeFormat(tsr.Start),
			"endTime":       timeFormat(tsr.End),
			"setupLogs":     tsr.SetupLogs,
			"triggeredBy":   tsr.TriggeredBy,
		})
	if err != nil {
		return -1, err
	}

	if !r.Next() {
		return -1, fmt.Errorf("retrieving inserted TestSuiteRun id")
	}

	var id int

	if err = r.Scan(&id); err != nil {
		return -1, fmt.Errorf("retrieving inserted TestSuiteRun id: %w", err)

	}

	return id, nil
}

func (s *Storage) UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	db := s.getDB(ctx)

	_, err := db.NamedExecContext(ctx, `UPDATE TestSuiteRun SET
	result=:result, startTime=:startTime, endTime=:endTime, setupLogs=:setupLogs
	where id = :id and suiteName = :suiteName`,
		map[string]any{
			"result":    tsr.Result,
			"startTime": timeFormat(tsr.Start),
			"endTime":   timeFormat(tsr.End),
			"setupLogs": tsr.SetupLogs,
			"id":        tsr.ID,
			"suiteName": tsr.SuiteName,
		})

	return err
}

func (s *Storage) LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	r, err := db.NamedQuery(`SELECT 
	id, suiteName, environment, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy
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

func (s *Storage) LoadPendingTestSuiteRuns(ctx context.Context) ([]model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	runs := []model.TestSuiteRun{}
	r, err := db.QueryxContext(ctx, `SELECT 
		id, suiteName, environment, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy
		FROM TestSuiteRun WHERE result='pending'`)
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

func (s *Storage) LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	runs := []model.TestSuiteRun{}
	r, err := db.NamedQuery(`SELECT 
		id, suiteName, environment, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy
		FROM TestSuiteRun WHERE suiteName=:suiteName`,
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

func (s *Storage) LoadTestRuns(ctx context.Context, suiteName string, tsrID int) ([]model.TestRun, error) {
	db := s.getDB(ctx)

	runs := []model.TestRun{}
	r, err := db.NamedQuery(`SELECT 
		suiteName, suiteRunId, testName, attempt, result, logs, context, startTime, endTime, softFailure
		FROM TestRun WHERE suiteName=:suiteName and suiteRunId=:suiteRunId`,
		map[string]any{
			"suiteRunId": tsrID,
			"suiteName":  suiteName,
		},
	)
	if err != nil {
		return runs, err
	}
	defer r.Close()

	for r.Next() {
		tr, err := scanTestRun(r)
		if err != nil {
			return nil, err
		}

		runs = append(runs, tr)
	}

	return runs, nil
}

func (s *Storage) LoadTestRun(ctx context.Context, suiteName string, tsrID int, testName string, attempt int) (model.TestRun, error) {
	db := s.getDB(ctx)

	r, err := db.NamedQuery(`SELECT 
		suiteName, suiteRunId, testName, attempt, result, logs, context, startTime, endTime, softFailure
		FROM TestRun WHERE suiteName=:suiteName and suiteRunId=:suiteRunId and testName=:testName and attempt=:attempt`,
		map[string]any{
			"suiteRunId": tsrID,
			"suiteName":  suiteName,
			"testName":   testName,
			"attempt":    attempt,
		},
	)
	if err != nil {
		return model.TestRun{}, err
	}
	defer r.Close()

	if !r.Next() {
		return model.TestRun{}, model.NotFoundError{}
	}

	return scanTestRun(r)
}

func (s *Storage) InsertTestRun(ctx context.Context, tr model.TestRun) error {
	testContext, err := json.Marshal(tr.Context)
	if err != nil {
		return fmt.Errorf("unable to marshal plugin data: %w", err)
	}

	db := s.getDB(ctx)
	_, err = db.NamedExecContext(ctx, `INSERT INTO TestRun
	(suiteName, suiteRunId, testName, result, logs, context, startTime, endTime, attempt, softFailure) VALUES
	(:suiteName, :suiteRunId, :testName, :result, :logs, :context, :startTime, :endTime, :attempt, :softFailure)`,
		map[string]any{
			"suiteName":   tr.SuiteName,
			"suiteRunId":  tr.SuiteRunID,
			"testName":    tr.Name,
			"result":      tr.Result,
			"logs":        tr.Logs,
			"context":     string(testContext),
			"startTime":   timeFormat(tr.Start),
			"endTime":     timeFormat(tr.End),
			"attempt":     tr.Attempt,
			"softFailure": tr.SoftFailure,
		})
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) InsertForcedTestRun(ctx context.Context, tr model.TestRun) (int, error) {
	testContext, err := json.Marshal(tr.Context)
	if err != nil {
		return -1, fmt.Errorf("unable to marshal plugin data: %w", err)
	}

	db := s.getDB(ctx)
	r, err := db.NamedQuery(`INSERT INTO TestRun
	(suiteName, suiteRunId, testName, result, logs, context, startTime, endTime, softFailure, attempt) VALUES
	(:suiteName, :suiteRunId, :testName, :result, :logs, :context, :startTime, :endTime, :softFailure,
		COALESCE(select max(attempt)+1 from TestRun where suiteName=:suiteName and suiteRunId=:suiteRunId and testName=:testName, 1)
	RETURNING attempt`,
		map[string]any{
			"suiteName":   tr.SuiteName,
			"suiteRunId":  tr.SuiteRunID,
			"testName":    tr.Name,
			"result":      tr.Result,
			"logs":        tr.Logs,
			"context":     string(testContext),
			"startTime":   timeFormat(tr.Start),
			"endTime":     timeFormat(tr.End),
			"softFailure": tr.SoftFailure,
		})
	if err != nil {
		return -1, err
	}
	defer r.Close()

	if !r.Next() {
		return -1, errors.New("could not retrieve created testrun attempt #")
	}

	var attempt int

	if err = r.Scan(attempt); err != nil {
		return -1, errors.New("could not retrieve created testrun attempt #")

	}

	return attempt, nil

}

func (s *Storage) UpdateTestRun(ctx context.Context, tr model.TestRun) error {
	testContext, err := json.Marshal(tr.Context)
	if err != nil {
		return fmt.Errorf("unable to marshal plugin data: %w", err)
	}

	db := s.getDB(ctx)
	r, err := db.NamedExecContext(ctx, `UPDATE TestRun SET
	result=:result, logs=:logs, startTime=:startTime, endTime=:endTime, context=:context, softFailure=:softFailure
	WHERE suiteName=:suiteName and suiteRunId=:suiteRunId and testName=:testName and attempt=:attempt`,
		map[string]any{
			"suiteName":   tr.SuiteName,
			"suiteRunId":  tr.SuiteRunID,
			"testName":    tr.Name,
			"attempt":     tr.Attempt,
			"result":      tr.Result,
			"logs":        tr.Logs,
			"context":     string(testContext),
			"startTime":   timeFormat(tr.Start),
			"endTime":     timeFormat(tr.End),
			"softFailure": tr.SoftFailure,
		})
	if err != nil {
		return fmt.Errorf("update statement failed")
	}

	if affected, _ := r.RowsAffected(); affected != 1 {
		return fmt.Errorf("test run not found")
	}

	return nil
}

func timeFormat(t time.Time) string {
	return t.Format(time.RFC3339)
}

func parseDate(t string) (time.Time, error) {
	return time.Parse(time.RFC3339, t)
}

func scanTestRun(r *sqlx.Rows) (model.TestRun, error) {
	tr := model.TestRun{}

	var start, end string

	var testContext []byte

	err := r.Scan(
		&tr.SuiteName,
		&tr.SuiteRunID,
		&tr.Name,
		&tr.Attempt,
		&tr.Result,
		&tr.Logs,
		&testContext,
		&start,
		&end,
		&tr.SoftFailure,
	)
	if err != nil {
		return model.TestRun{}, fmt.Errorf("scanning test suite run: %w", err)
	}

	if tr.Start, err = parseDate(start); err != nil {
		return model.TestRun{}, fmt.Errorf("parsing start time: %w", err)
	}
	if tr.End, err = parseDate(end); err != nil {
		return model.TestRun{}, fmt.Errorf("parsing end time: %w", err)
	}
	if err = json.Unmarshal(testContext, &tr.Context); err != nil {
		return model.TestRun{}, fmt.Errorf("unmarshaling plugin data: %w", err)
	}

	tr.DurationInMS = tr.End.Sub(tr.Start).Milliseconds()

	return tr, nil
}

func scanTestSuiteRun(r *sqlx.Rows) (model.TestSuiteRun, error) {
	tsr := model.TestSuiteRun{}

	var start, end, scheduled string

	err := r.Scan(
		&tsr.ID,
		&tsr.SuiteName,
		&tsr.Environment,
		&tsr.Result,
		&tsr.TestFilter,
		&scheduled,
		&start,
		&end,
		&tsr.SetupLogs,
		&tsr.TriggeredBy,
	)
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("scanning test suite run: %w", err)
	}

	if tsr.Start, err = parseDate(start); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing start time: %w", err)
	}
	if tsr.End, err = parseDate(end); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing end time: %w", err)
	}
	if tsr.Scheduled, err = parseDate(scheduled); err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("parsing scheduled time: %w", err)
	}

	tsr.DurationInMS = tsr.End.Sub(tsr.Start).Milliseconds()

	if tsr.TestFilter != "" {
		tsr.TestFilterRegex, err = regexp.Compile(tsr.TestFilter)
		if err != nil {
			return model.TestSuiteRun{}, fmt.Errorf("compiling test filter regex %s: %w", tsr.TestFilter, err)
		}
	}

	return tsr, nil
}
