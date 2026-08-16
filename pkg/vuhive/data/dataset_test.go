package data_test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive/pkg/vuhive/data"
)

type mockContext struct {
	vuid      int64
	iteration int64
}

func (m mockContext) VUID() int64      { return m.vuid }
func (m mockContext) Iteration() int64 { return m.iteration }

// AC-1.14.1: LoadCSV parses CSV with headers into string key-value maps
func TestLoadCSV(t *testing.T) {
	csvData := `username,role,user_id
alice,admin,101
bob,user,102`

	ds, err := data.LoadCSV(bytes.NewBufferString(csvData), data.Sequential)
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.Equal(t, 2, ds.Len())

	rec1, err := ds.Next(mockContext{vuid: 1, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "alice", rec1["username"])
	assert.Equal(t, "admin", rec1["role"])
	assert.Equal(t, "101", rec1["user_id"])

	rec2, err := ds.Next(mockContext{vuid: 1, iteration: 1})
	require.NoError(t, err)
	assert.Equal(t, "bob", rec2["username"])
	assert.Equal(t, "user", rec2["role"])
	assert.Equal(t, "102", rec2["user_id"])
}

// AC-1.14.2: LoadJSON parses JSON array of objects into key-value maps
func TestLoadJSON(t *testing.T) {
	jsonData := `[
		{"username": "alice", "status_code": 200, "active": true},
		{"username": "bob", "status_code": 404, "active": false}
	]`

	ds, err := data.LoadJSON(bytes.NewBufferString(jsonData), data.Sequential)
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.Equal(t, 2, ds.Len())

	rec1, err := ds.Next(mockContext{vuid: 1, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "alice", rec1["username"])
	assert.Equal(t, "200", rec1["status_code"])
	assert.Equal(t, "true", rec1["active"])

	rec2, err := ds.Next(mockContext{vuid: 1, iteration: 1})
	require.NoError(t, err)
	assert.Equal(t, "bob", rec2["username"])
	assert.Equal(t, "404", rec2["status_code"])
	assert.Equal(t, "false", rec2["active"])
}

// AC-1.14.3: LoadJSONL parses newline-delimited JSON objects
func TestLoadJSONL(t *testing.T) {
	jsonlData := `{"event": "login", "user_id": "u101"}
{"event": "checkout", "user_id": "u102"}
`

	ds, err := data.LoadJSONL(bytes.NewBufferString(jsonlData), data.Sequential)
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.Equal(t, 2, ds.Len())

	rec1, err := ds.Next(mockContext{vuid: 1, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "login", rec1["event"])
	assert.Equal(t, "u101", rec1["user_id"])

	rec2, err := ds.Next(mockContext{vuid: 1, iteration: 1})
	require.NoError(t, err)
	assert.Equal(t, "checkout", rec2["event"])
	assert.Equal(t, "u102", rec2["user_id"])
}

// AC-1.14.4: Sequential strategy round-robins across rows deterministically by VU ID and iteration
func TestStrategySequential(t *testing.T) {
	records := []data.Record{
		{"id": "0"},
		{"id": "1"},
		{"id": "2"},
	}
	ds := data.NewDataSet(records, data.Sequential)

	// VU 1: iter 0 -> row 0, iter 1 -> row 1, iter 2 -> row 2, iter 3 -> row 0
	r0, err := ds.Next(mockContext{vuid: 1, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "0", r0["id"])

	r1, err := ds.Next(mockContext{vuid: 1, iteration: 1})
	require.NoError(t, err)
	assert.Equal(t, "1", r1["id"])

	r2, err := ds.Next(mockContext{vuid: 1, iteration: 2})
	require.NoError(t, err)
	assert.Equal(t, "2", r2["id"])

	r3, err := ds.Next(mockContext{vuid: 1, iteration: 3})
	require.NoError(t, err)
	assert.Equal(t, "0", r3["id"])

	// VU 2: iter 0 -> row 1
	rVU2, err := ds.Next(mockContext{vuid: 2, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "1", rVU2["id"])
}

// AC-1.14.5: Random strategy selects rows uniformly with thread safety
func TestStrategyRandom(t *testing.T) {
	records := make([]data.Record, 10)
	for i := range 10 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Random)

	const totalCalls = 100
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var mu sync.Mutex
	counts := make(map[string]int)

	for g := range goroutines {
		go func(vuid int) {
			defer wg.Done()
			for iter := range totalCalls / goroutines {
				rec, err := ds.Next(mockContext{vuid: int64(vuid + 1), iteration: int64(iter)})
				assert.NoError(t, err)
				mu.Lock()
				counts[rec["id"]]++
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	assert.Len(t, counts, 10, "random selection should touch all records over 100 calls")
}

// AC-1.14.6: SharedQueue strategy dispenses each row exactly once across concurrent VUs
func TestStrategySharedQueue(t *testing.T) {
	const totalRecords = 50
	records := make([]data.Record, totalRecords)
	for i := range totalRecords {
		records[i] = data.Record{"id": fmt.Sprintf("item-%d", i)}
	}

	ds := data.NewDataSet(records, data.SharedQueue)

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var mu sync.Mutex
	dispensed := make(map[string]int)

	for g := range goroutines {
		go func(vuid int) {
			defer wg.Done()
			for iter := 0; ; iter++ {
				rec, err := ds.Next(mockContext{vuid: int64(vuid + 1), iteration: int64(iter)})
				if errors.Is(err, data.ErrDatasetExhausted) {
					break
				}
				assert.NoError(t, err)
				mu.Lock()
				dispensed[rec["id"]]++
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	assert.Len(t, dispensed, totalRecords, "every item must be dispensed")
	for id, count := range dispensed {
		assert.Equal(t, 1, count, "item %s must be dispensed exactly once", id)
	}

	// Further calls must return ErrDatasetExhausted
	_, err := ds.Next(mockContext{vuid: 1, iteration: 99})
	assert.ErrorIs(t, err, data.ErrDatasetExhausted)
}

func TestStrategyUniquePerVU(t *testing.T) {
	records := []data.Record{
		{"id": "0"},
		{"id": "1"},
		{"id": "2"},
		{"id": "3"},
	}
	ds := data.NewDataSet(records, data.UniquePerVU)

	// VU 1 -> row 0
	r1, err := ds.Next(mockContext{vuid: 1, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "0", r1["id"])

	// VU 2 -> row 1
	r2, err := ds.Next(mockContext{vuid: 2, iteration: 0})
	require.NoError(t, err)
	assert.Equal(t, "1", r2["id"])
}

func TestDataSet_Next_NilContextValidation(t *testing.T) {
	records := []data.Record{
		{"id": "0"},
		{"id": "1"},
	}

	t.Run("Sequential returns ErrNilContext when ctx is nil", func(t *testing.T) {
		ds := data.NewDataSet(records, data.Sequential)
		rec, err := ds.Next(nil)
		assert.Nil(t, rec)
		assert.ErrorIs(t, err, data.ErrNilContext)
	})

	t.Run("UniquePerVU returns ErrNilContext when ctx is nil", func(t *testing.T) {
		ds := data.NewDataSet(records, data.UniquePerVU)
		rec, err := ds.Next(nil)
		assert.Nil(t, rec)
		assert.ErrorIs(t, err, data.ErrNilContext)
	})

	t.Run("Random succeeds when ctx is nil", func(t *testing.T) {
		ds := data.NewDataSet(records, data.Random)
		rec, err := ds.Next(nil)
		assert.NoError(t, err)
		assert.NotNil(t, rec)
	})

	t.Run("SharedQueue succeeds when ctx is nil", func(t *testing.T) {
		ds := data.NewDataSet(records, data.SharedQueue)
		rec, err := ds.Next(nil)
		assert.NoError(t, err)
		assert.NotNil(t, rec)
	})
}

func TestStrategyRandom_HighConcurrency(t *testing.T) {
	records := make([]data.Record, 20)
	for i := range 20 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Random)

	const goroutines = 50
	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(vuid int) {
			defer wg.Done()
			for iter := range iterations {
				rec, err := ds.Next(mockContext{vuid: int64(vuid + 1), iteration: int64(iter)})
				assert.NoError(t, err)
				assert.NotEmpty(t, rec["id"])
			}
		}(g)
	}

	wg.Wait()
}

func BenchmarkStrategyRandom(b *testing.B) {
	records := make([]data.Record, 100)
	for i := range 100 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Random)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := mockContext{vuid: 1, iteration: 1}
		for pb.Next() {
			_, _ = ds.Next(ctx)
		}
	})
}

func BenchmarkStrategySequential(b *testing.B) {
	records := make([]data.Record, 100)
	for i := range 100 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Sequential)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := mockContext{vuid: 1, iteration: 1}
		for pb.Next() {
			_, _ = ds.Next(ctx)
		}
	})
}


