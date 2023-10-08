package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"github.com/raphi011/handoff/internal/model"
)

type BadgerStorage struct {
	db  *badger.DB
	log *slog.Logger

	lock      sync.Mutex
	sequences map[string]*badger.Sequence
}

func NewBadgerStorage(dbPath string, log *slog.Logger) (*BadgerStorage, error) {
	badgerDB, err := badger.Open(badger.DefaultOptions(dbPath).
		WithLoggingLevel(badger.ERROR).
		WithInMemory(dbPath == ""))
	if err != nil {
		return nil, fmt.Errorf("opening badger database: %w", err)
	}

	s := &BadgerStorage{
		log:       log,
		db:        badgerDB,
		sequences: make(map[string]*badger.Sequence),
	}

	return s, nil
}

func (b *BadgerStorage) Close() error {
	return b.db.Close()
}

type BadgerStorageContextKey string

func (b *BadgerStorage) StartTransaction(ctx context.Context) (context.Context, error) {
	txn := b.db.NewTransaction(true)

	return context.WithValue(ctx, BadgerStorageContextKey("transaction"), txn), nil
}

func getTx(ctx context.Context) *badger.Txn {
	v := ctx.Value(BadgerStorageContextKey("transaction"))

	tx, _ := v.(*badger.Txn)

	return tx
}

func (b *BadgerStorage) runTx(ctx context.Context, ftx func(t *badger.Txn) error) error {
	if tx := getTx(ctx); tx != nil {
		return ftx(tx)
	} else {
		return b.db.Update(ftx)
	}
}

func (b *BadgerStorage) CommitTransaction(ctx context.Context) error {
	return getTx(ctx).Commit()
}

func (b *BadgerStorage) RollbackTransaction(ctx context.Context) {
	getTx(ctx).Discard()
}

func testSuiteRunKey(suiteName string, id int) []byte {
	return []byte(fmt.Sprintf("%s-%v", suiteName, id))
}

func (b *BadgerStorage) getSequence(key []byte) (*badger.Sequence, error) {
	b.lock.Lock()
	seq, found := b.sequences[string(key)]
	b.lock.Unlock()

	if found {
		return seq, nil
	}

	b.lock.Lock()

	seq, found = b.sequences[string(key)]
	if !found {
		var err error
		seq, err = b.db.GetSequence(key, 1000)
		if err != nil {
			return nil, fmt.Errorf("unable to get sequence: %w", err)
		}
		b.sequences[string(key)] = seq
	}

	b.lock.Unlock()

	return seq, nil
}

func (b *BadgerStorage) InsertTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) (int, error) {
	seq, err := b.getSequence([]byte(tsr.SuiteName))
	if err != nil {
		return -1, err
	}

	seqID, err := seq.Next()
	if err != nil {
		return -1, fmt.Errorf("unable to get next sequence: %w", err)
	}
	id := int(seqID) + 1
	// start sequence with 1
	tsr.ID = id

	for _, t := range tsr.TestResults {
		t.SuiteRunID = id
	}

	return id, b.UpdateTestSuiteRun(ctx, tsr)
}

func (b *BadgerStorage) UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	err := b.runTx(ctx, func(t *badger.Txn) error {
		data, err := json.Marshal(tsr)
		if err != nil {
			return fmt.Errorf("marshalling test suite run: %w", err)
		}

		err = t.Set(testSuiteRunKey(tsr.SuiteName, tsr.ID), data)
		if err != nil {
			return fmt.Errorf("inserting test suite run: %w", err)
		}

		return nil
	})

	return err
}

func (b *BadgerStorage) LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error) {
	var tsr model.TestSuiteRun

	err := b.runTx(ctx, func(txn *badger.Txn) error {
		item, err := txn.Get(testSuiteRunKey(suiteName, runID))
		if err != nil {
			return fmt.Errorf("loading test suite run: %w", err)
		}

		err = item.Value(func(d []byte) error {
			return json.Unmarshal(d, &tsr)
		})
		if err != nil {
			return fmt.Errorf("unmarshaling test suite run: %w", err)
		}

		return nil
	})

	return tsr, err
}

func (b *BadgerStorage) LoadPendingTestSuiteRuns(ctx context.Context) ([]model.TestSuiteRun, error) {
	// todo
	return []model.TestSuiteRun{}, nil
}

func (b *BadgerStorage) LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error) {
	runs := []model.TestSuiteRun{}

	err := b.runTx(ctx, func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(suiteName + "-")

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var tsr model.TestSuiteRun

			err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &tsr)
			})

			if err != nil {
				return fmt.Errorf("unmarshaling test suite run: %w", err)
			}

			runs = append(runs, tsr)
		}

		return nil
	})

	return runs, err
}
