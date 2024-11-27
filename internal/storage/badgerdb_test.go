package storage_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/raphi011/handoff/internal/model"
	"github.com/raphi011/handoff/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestBadgerdb(t *testing.T) {
	db, err := storage.NewBadgerStorage("", 0, nil, slog.Default())
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

func TestDeleteScheduledRun(t *testing.T) {
	db, err := storage.NewBadgerStorage("", 0, nil, slog.Default())
	assert.NoError(t, err)

	ctx := context.Background()

	assert.NoError(t, db.InsertScheduledRun(ctx, model.ScheduledRun{
		Name:          "sr",
		TestSuiteName: "sn",
	}))

	err = db.DeleteScheduledRun(ctx, "sr")
	assert.NoError(t, err)
}

func TestIdempotencyKey(t *testing.T) {
	db, err := storage.NewBadgerStorage("", 0, nil, slog.Default())
	assert.NoError(t, err)

	ctx := context.Background()

	tsrID, err := db.InsertIdempotencyKey(ctx, "foo")
	assert.NoError(t, err)
	assert.Empty(t, tsrID)

	var duplicateErr model.DuplicateError

	tsrID, err = db.InsertIdempotencyKey(ctx, "foo")
	assert.ErrorAs(t, err, &duplicateErr)
	assert.Empty(t, tsrID)

	err = db.UpdateIdempotencyKey(ctx, "foo", "my-app", 1)
	assert.NoError(t, err)

	tsrID, err = db.InsertIdempotencyKey(ctx, "foo")
	assert.ErrorAs(t, err, &duplicateErr)
	assert.Equal(t, tsrID, "suite-my-app-1")
}

func TestTransaction(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").
		WithLoggingLevel(badger.ERROR).
		WithInMemory(true))

	assert.NoError(t, err)

	defer db.Close()

	var mu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)

	goroutineTask := func(id int, key, value string) {
		defer wg.Done() // Mark this goroutine as done when it completes

		err := db.Update(func(txn *badger.Txn) error {
			mu.Lock()

			if _, err := txn.Get([]byte(key)); err == badger.ErrKeyNotFound {
				fmt.Printf("%d: key not found\n", id)
			} else {
				fmt.Printf("%d: key found\n", id)
			}
			mu.Unlock()
			time.Sleep(500 * time.Millisecond)

			mu.Lock()
			err := txn.Set([]byte(key), []byte(value))
			if err != nil {
				return err
			}
			mu.Unlock()
			return nil
		})

		if err != nil {
			log.Printf("%d: Failed to write key '%s': %v\n", id, key, err)
		}
	}

	// Start two goroutines that each try to write to the database
	go goroutineTask(1, "key", "value1")
	go goroutineTask(2, "key2", "value2")

	// Wait for both goroutines to finish
	wg.Wait()

	fmt.Println("All goroutines have finished.")

}
