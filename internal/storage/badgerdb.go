package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/raphi011/handoff/internal/model"
	"github.com/robfig/cron/v3"
)

type BadgerStorage struct {
	db  *badger.DB
	log *slog.Logger
	ttl time.Duration

	lock      sync.Mutex
	sequences map[string]*badger.Sequence
}

func NewBadgerStorage(dbPath string,
	ttl time.Duration,
	gcCron *cron.Cron,
	log *slog.Logger,
) (*BadgerStorage, error) {
	badgerDB, err := badger.Open(badger.DefaultOptions(dbPath).
		WithLoggingLevel(badger.ERROR).
		WithInMemory(dbPath == ""))
	if err != nil {
		return nil, fmt.Errorf("opening badger database: %w", err)
	}

	if gcCron != nil && dbPath != "" {
		_, err = gcCron.AddFunc("@every 5m", func() {
			if err := badgerDB.RunValueLogGC(0.5); err != nil && err != badger.ErrNoRewrite {
				log.Warn("badger gc", "error", err)
			}
		})
		if err != nil {
			return nil, err
		}
	}

	s := &BadgerStorage{
		log:       log,
		db:        badgerDB,
		ttl:       ttl,
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

func (b *BadgerStorage) runTx(ctx context.Context, write bool, ftx func(t *badger.Txn) error) error {
	if tx := getTx(ctx); tx != nil {
		return ftx(tx)
	} else {
		if write {
			return b.db.Update(ftx)
		} else {
			return b.db.View(ftx)
		}
	}
}

func (b *BadgerStorage) CommitTransaction(ctx context.Context) error {
	return getTx(ctx).Commit()
}

func (b *BadgerStorage) RollbackTransaction(ctx context.Context) {
	getTx(ctx).Discard()
}

func testSuiteRunKey(suiteName string, id int) []byte {
	return []byte(fmt.Sprintf("suite-%s-%v", suiteName, id))
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
		seq, err = b.db.GetSequence(key, 1)
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

	err = b.runTx(ctx, true, func(t *badger.Txn) error {
		data, err := json.Marshal(tsr)
		if err != nil {
			return fmt.Errorf("marshalling test suite run: %w", err)
		}

		key := testSuiteRunKey(tsr.SuiteName, tsr.ID)
		e := badger.NewEntry(key, data)
		// TODO: maybe set run status as `UserMeta`?

		err = t.SetEntry(e)
		if err != nil {
			return fmt.Errorf("inserting test suite run: %w", err)
		}

		pendingKey := append([]byte("pending-"), key...)

		err = t.Set(pendingKey, nil)
		if err != nil {
			return fmt.Errorf("add pending key: %w", err)
		}

		return nil
	})

	return id, err
}

func (b *BadgerStorage) UpdateTestSuiteRun(ctx context.Context, tsr model.TestSuiteRun) error {
	err := b.runTx(ctx, true, func(t *badger.Txn) error {
		data, err := json.Marshal(tsr)
		if err != nil {
			return fmt.Errorf("marshalling test suite run: %w", err)
		}

		key := testSuiteRunKey(tsr.SuiteName, tsr.ID)
		pendingKey := append([]byte("pending-"), key...)

		e := badger.NewEntry(key, data)

		if tsr.Result != model.ResultPending {
			err = t.Delete(pendingKey)
			if err != nil {
				return fmt.Errorf("deleting pending key: %w", err)
			}

			if b.ttl > 0 {
				e = e.WithTTL(b.ttl)
			}
		}

		err = t.SetEntry(e)
		if err != nil {
			return fmt.Errorf("inserting test suite run: %w", err)
		}

		return nil
	})

	return err
}

func (b *BadgerStorage) LoadTestSuiteRunByKey(
	ctx context.Context,
	key string,
) (model.TestSuiteRun, error) {
	parts := strings.Split(key, "-")

	if len(parts) < 3 {
		return model.TestSuiteRun{}, fmt.Errorf("invalid testsuite run key: %s", key)
	}

	suiteName := strings.Join(parts[1:len(parts)-1], "-")
	runID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return model.TestSuiteRun{}, fmt.Errorf("id part of the key %s it not an int", key)
	}

	return b.LoadTestSuiteRun(ctx, suiteName, runID)
}

func (b *BadgerStorage) LoadTestSuiteRun(ctx context.Context, suiteName string, runID int) (model.TestSuiteRun, error) {
	var tsr model.TestSuiteRun

	err := b.runTx(ctx, false, func(txn *badger.Txn) error {
		var err error
		tsr, err = loadTestSuiteRun(txn, testSuiteRunKey(suiteName, runID))
		return err
	})

	return tsr, err
}

func loadTestSuiteRun(txn *badger.Txn, key []byte) (model.TestSuiteRun, error) {
	var tsr model.TestSuiteRun

	item, err := txn.Get(key)
	if err != nil {
		return tsr, fmt.Errorf("loading test suite run: %w", err)
	}

	err = item.Value(func(d []byte) error {
		return json.Unmarshal(d, &tsr)
	})
	if err != nil {
		return tsr, fmt.Errorf("unmarshaling test suite run: %w", err)
	}

	return tsr, nil
}

func (b *BadgerStorage) LoadPendingTestSuiteRuns(ctx context.Context) ([]model.TestSuiteRun, error) {
	pending := []model.TestSuiteRun{}

	err := b.runTx(ctx, false, func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("pending-")

		keys := []string{}

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := string(item.Key())

			suiteKey := k[len("pending-"):]

			keys = append(keys, suiteKey)
		}

		for _, suiteKey := range keys {
			tsr, err := loadTestSuiteRun(txn, []byte(suiteKey))
			if err != nil {
				return err
			}
			pending = append(pending, tsr)
		}

		return nil
	})

	return pending, err
}

func (b *BadgerStorage) LoadTestSuiteRunsByName(ctx context.Context, suiteName string) ([]model.TestSuiteRun, error) {
	runs := []model.TestSuiteRun{}

	err := b.runTx(ctx, false, func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte("suite-" + suiteName + "-")

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

func scheduledRunKey(scheduleName string) []byte {
	return []byte("schedule-" + scheduleName)
}

func (b *BadgerStorage) InsertScheduledRun(ctx context.Context, sr model.ScheduledRun) error {
	err := b.runTx(ctx, true, func(t *badger.Txn) error {
		data, err := json.Marshal(sr)
		if err != nil {
			return fmt.Errorf("marshalling scheduled run: %w", err)
		}

		key := scheduledRunKey(sr.Name)
		e := badger.NewEntry(key, data)

		err = t.SetEntry(e)
		if err != nil {
			return fmt.Errorf("inserting scheduled run: %w", err)
		}

		return nil
	})

	return err
}

func (b *BadgerStorage) DeleteScheduledRun(ctx context.Context, name string) error {
	err := b.runTx(ctx, true, func(t *badger.Txn) error {
		_, err := t.Get(scheduledRunKey(name))
		if err == badger.ErrKeyNotFound {
			return model.NotFoundError{}
		} else if err != nil {
			return err
		}

		return t.Delete(scheduledRunKey(name))
	})

	if err != nil {
		return fmt.Errorf("deleting scheduled run: %w", err)
	}

	return nil
}

func idempotencyKey(key string) []byte {
	return []byte("idempotent-" + key)
}

func (b *BadgerStorage) InsertIdempotencyKey(ctx context.Context, key string) (string, error) {
	tsrID := ""

	getTsrId := func(entry *badger.Item) error {
		_ = entry.Value(func(val []byte) error {
			tsrID = string(val)
			return nil
		})
		return model.DuplicateError{}
	}

	k := idempotencyKey(key)

	err := b.runTx(ctx, true, func(t *badger.Txn) error {

		entry, err := t.Get(k)

		if err == badger.ErrKeyNotFound {
			return t.Set(k, nil)
		} else if err != nil {
			return err
		} else {
			return getTsrId(entry)
		}
	})

	if err == badger.ErrConflict {
		// try once more to get the value
		err = b.runTx(ctx, false, func(t *badger.Txn) error {
			if entry, err := t.Get(k); err == nil {
				return getTsrId(entry)
			} else {
				return err
			}
		})
	}

	return tsrID, err
}

func (b *BadgerStorage) UpdateIdempotencyKey(
	ctx context.Context,
	key string,
	suiteName string,
	id int,
) error {
	runKey := testSuiteRunKey(suiteName, id)
	k := idempotencyKey(key)

	return b.runTx(ctx, true, func(t *badger.Txn) error {
		return t.SetEntry(badger.NewEntry(k, runKey).WithTTL(72 * time.Hour))
	})
}
