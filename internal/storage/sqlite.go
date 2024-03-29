package storage

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"regexp"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/raphi011/handoff/internal/model"
)

//go:embed migrations/*.sql
var fs embed.FS

type SqliteStorage struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewSqlite(dbFilename string, log *slog.Logger) (*SqliteStorage, error) {
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

	log.Info("Using sqlite version: " + version)

	s := &SqliteStorage{
		db:  db,
		log: log,
	}

	if err = s.migrateDB(db); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SqliteStorage) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("closing DB connection: %w", err)
	}

	return nil
}

func connectionString(filename string) string {
	var cs string
	var options = []string{
		"_pragma=busy_timeout(5000)",
		"_pragma=journal_mode(WAL)",
		"_pragma=journal_size_limit(6144000)",
		"_pragma=foreign_keys(1)",
		"_pragma=synchronous(normal)",
		"_pragma=mmap_size(30000000000)",
		"_pragma=page_size(32768)",
	}

	if filename != "" {
		cs = filename
	} else {
		cs = "file:" + randomAlphanumeric(16)
		options = append(options, "mode=memory", "cache=shared")
	}

	for i, o := range options {
		if i == 0 {
			cs += "?"
		} else {
			cs += "&"
		}
		cs += o
	}

	return cs
}

const alphaNumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomAlphanumeric(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphaNumericChars[rand.Intn(len(alphaNumericChars))]
	}
	return string(b)
}

func (s *SqliteStorage) migrateDB(db *sqlx.DB) error {
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
		s.log.Info("No migrations have been applied. The DB is at the latest state.")
	} else if err != nil {
		return fmt.Errorf("applying db migrations: %w", err)
	}

	return nil
}

type SqliteStorageContextKey string

func (s *SqliteStorage) StartTransaction(ctx context.Context) (context.Context, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, SqliteStorageContextKey("SqliteStorage.transaction"), tx), nil
}

func (s *SqliteStorage) CommitTransaction(ctx context.Context) error {
	v := ctx.Value(SqliteStorageContextKey("SqliteStorage.transaction"))

	if v == nil {
		return errors.New("context does not contain a transaction")
	}

	return v.(*sqlx.Tx).Commit()
}

func (s *SqliteStorage) RollbackTransaction(ctx context.Context) {
	v := ctx.Value(SqliteStorageContextKey("SqliteStorage.transaction"))

	if v != nil {
		err := v.(*sqlx.Tx).Rollback()
		if err != nil && err != sql.ErrTxDone {
			s.log.Warn("could not rollback transaction", "error", err)
		}
	}
}

func (s *SqliteStorage) getDB(ctx context.Context) commonDB {
	v := ctx.Value(SqliteStorageContextKey("SqliteStorage.transaction"))

	if v == nil {
		return s.db
	}

	return v.(*sqlx.Tx)
}

// functions shared by `*sqlx.Tx` and `*sqlx.Db`
type commonDB interface {
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
}

func (s *SqliteStorage) InsertTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	db := s.getDB(ctx)

	testFilter := ""

	if tsr.Params.TestFilter != nil {
		testFilter = tsr.Params.TestFilter.String()
	}

	// r, err := db.NamedQuery(`INSERT INTO TestSuiteRun
	// (suiteName, environment, reference, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, maxTestAttempts, timeoutDuration, flaky, id) VALUES
	// (:suiteName, :environment, :reference, :result, :testFilter, :scheduledTime, :startTime, :endTime, :setupLogs, :triggeredBy, :maxTestAttempts, :timeoutDuration, :flaky,
	// 	COALESCE((select max(id)+1 from TestSuiteRun where suiteName=:suiteName), 1))
	// RETURNING id`,
	_, err := db.NamedExecContext(ctx, `INSERT INTO TestSuiteRun 
	(id, suiteName, environment, reference, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, maxTestAttempts, timeoutDuration, flaky) VALUES
	(:id, :suiteName, :environment, :reference, :result, :testFilter, :scheduledTime, :startTime, :endTime, :setupLogs, :triggeredBy, :maxTestAttempts, :timeoutDuration, :flaky)`,
		map[string]any{
			"id":              tsr.ID,
			"suiteName":       tsr.SuiteName,
			"environment":     tsr.Environment,
			"result":          tsr.Result,
			"scheduledTime":   timeFormat(tsr.Scheduled),
			"startTime":       timeFormat(tsr.Start),
			"endTime":         timeFormat(tsr.End),
			"flaky":           tsr.Flaky,
			"setupLogs":       tsr.SetupLogs,
			"triggeredBy":     tsr.Params.TriggeredBy,
			"reference":       tsr.Params.Reference,
			"maxTestAttempts": tsr.Params.MaxTestAttempts,
			"testFilter":      testFilter,
			"timeoutDuration": tsr.Params.Timeout,
		})
	if err != nil {
		return err
	}

	return nil
}

func (s *SqliteStorage) UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	db := s.getDB(ctx)

	_, err := db.NamedExecContext(ctx, `UPDATE TestSuiteRun SET
	result=:result, startTime=:startTime, endTime=:endTime, setupLogs=:setupLogs, flaky=:flaky
	where id = :id and suiteName = :suiteName`,
		map[string]any{
			"result":    tsr.Result,
			"startTime": timeFormat(tsr.Start),
			"endTime":   timeFormat(tsr.End),
			"setupLogs": tsr.SetupLogs,
			"id":        tsr.ID,
			"suiteName": tsr.SuiteName,
			"flaky":     tsr.Flaky,
		})

	return err
}

func (s *SqliteStorage) LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	r, err := db.NamedQuery(`SELECT 
	id, suiteName, environment, reference, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, maxTestAttempts, timeoutDuration
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

func (s *SqliteStorage) LoadPendingTestSuiteRuns(ctx context.Context) ([]model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	runs := []model.TestSuiteRun{}
	r, err := db.QueryxContext(ctx, `SELECT 
		id, suiteName, environment, reference, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, maxTestAttempts, timeoutDuration
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

func (s *SqliteStorage) LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error) {
	db := s.getDB(ctx)

	runs := []model.TestSuiteRun{}
	r, err := db.NamedQuery(`SELECT 
		id, suiteName, environment, reference, result, testFilter, scheduledTime, startTime, endTime, setupLogs, triggeredBy, maxTestAttempts, timeoutDuration
		FROM TestSuiteRun WHERE suiteName=:suiteName
		ORDER BY id ASC`,
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

func (s *SqliteStorage) LoadTestRuns(ctx context.Context, suiteName string, tsrID int) ([]*model.TestRun, error) {
	db := s.getDB(ctx)

	runs := []*model.TestRun{}
	r, err := db.NamedQuery(`SELECT 
		suiteName, suiteRunId, testName, attempt, result, compressedLogs, context, startTime, endTime, softFailure
		FROM TestRun WHERE suiteName=:suiteName and suiteRunId=:suiteRunId
		ORDER BY testName, attempt`,
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

		runs = append(runs, &tr)
	}

	return runs, nil
}

func (s *SqliteStorage) LoadTestRun(ctx context.Context, suiteName string, tsrID int, testName string) ([]model.TestRun, error) {
	db := s.getDB(ctx)

	tests := []model.TestRun{}
	r, err := db.NamedQuery(`SELECT 
		suiteName, suiteRunId, testName, attempt, result, compressedLogs, context, startTime, endTime, softFailure
		FROM TestRun WHERE suiteName=:suiteName and suiteRunId=:suiteRunId and testName=:testName
		ORDER BY attempt ASC`,
		map[string]any{
			"suiteRunId": tsrID,
			"suiteName":  suiteName,
			"testName":   testName,
		},
	)
	if err != nil {
		return tests, err
	}
	defer r.Close()

	for r.Next() {
		tsr, err := scanTestRun(r)
		if err != nil {
			return tests, err
		}

		tests = append(tests, tsr)
	}

	return tests, nil
}

func (s *SqliteStorage) InsertTestRun(ctx context.Context, tr model.TestRun) error {
	testContext, err := json.Marshal(tr.Context)
	if err != nil {
		return fmt.Errorf("unable to marshal plugin data: %w", err)
	}

	logs, err := compressedLogs(tr.Logs)
	if err != nil {
		return fmt.Errorf("unable to compress logs: %w", err)
	}

	db := s.getDB(ctx)
	_, err = db.NamedExecContext(ctx, `INSERT INTO TestRun
	(suiteName, suiteRunId, testName, result, compressedLogs, context, startTime, endTime, attempt, softFailure) VALUES
	(:suiteName, :suiteRunId, :testName, :result, :logs, :context, :startTime, :endTime, :attempt, :softFailure)`,
		map[string]any{
			"suiteName":   tr.SuiteName,
			"suiteRunId":  tr.SuiteRunID,
			"testName":    tr.Name,
			"result":      tr.Result,
			"logs":        logs,
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

func (s *SqliteStorage) UpdateTestRun(ctx context.Context, tr model.TestRun) error {
	testContext, err := json.Marshal(tr.Context)
	if err != nil {
		return fmt.Errorf("unable to marshal plugin data: %w", err)
	}

	logs, err := compressedLogs(tr.Logs)
	if err != nil {
		return fmt.Errorf("unable to compress logs: %w", err)
	}

	db := s.getDB(ctx)
	r, err := db.NamedExecContext(ctx, `UPDATE TestRun SET
	result=:result, compressedLogs=:logs, startTime=:startTime, endTime=:endTime, context=:context, softFailure=:softFailure
	WHERE suiteName=:suiteName and suiteRunId=:suiteRunId and testName=:testName and attempt=:attempt`,
		map[string]any{
			"suiteName":   tr.SuiteName,
			"suiteRunId":  tr.SuiteRunID,
			"testName":    tr.Name,
			"attempt":     tr.Attempt,
			"result":      tr.Result,
			"logs":        logs,
			"context":     string(testContext),
			"startTime":   timeFormat(tr.Start),
			"endTime":     timeFormat(tr.End),
			"softFailure": tr.SoftFailure,
		})
	if err != nil {
		return fmt.Errorf("update statement failed: %w", err)
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

	var logs []byte

	err := r.Scan(
		&tr.SuiteName,
		&tr.SuiteRunID,
		&tr.Name,
		&tr.Attempt,
		&tr.Result,
		&logs,
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

	tr.Logs, err = decompressLogs(logs)
	if err != nil {
		return model.TestRun{}, err
	}

	tr.DurationInMS = tr.End.Sub(tr.Start).Milliseconds()

	return tr, nil
}

func compressedLogs(logs string) ([]byte, error) {
	var compressedLogs bytes.Buffer

	w := zlib.NewWriter(&compressedLogs)

	_, err := w.Write([]byte(logs))
	w.Close()

	return compressedLogs.Bytes(), err
}

func decompressLogs(l []byte) (string, error) {
	if len(l) == 0 {
		return "", nil
	}

	reader, err := zlib.NewReader(bytes.NewReader(l))
	if err != nil {
		return "", fmt.Errorf("decompress logs: %w", err)
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("decompress logs: %w", err)
	}

	return string(logs), nil
}

func scanTestSuiteRun(r *sqlx.Rows) (model.TestSuiteRun, error) {
	tsr := model.TestSuiteRun{}

	var start, end, scheduled string

	var testFilter string

	err := r.Scan(
		&tsr.ID,
		&tsr.SuiteName,
		&tsr.Environment,
		&tsr.Params.Reference,
		&tsr.Result,
		&testFilter,
		&scheduled,
		&start,
		&end,
		&tsr.SetupLogs,
		&tsr.Params.TriggeredBy,
		&tsr.Params.MaxTestAttempts,
		&tsr.Params.Timeout,
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

	if testFilter != "" {
		tsr.Params.TestFilter, err = regexp.Compile(testFilter)
		if err != nil {
			return model.TestSuiteRun{}, fmt.Errorf("compiling test filter regex %s: %w", tsr.Params.TestFilter, err)
		}
	}

	return tsr, nil
}
