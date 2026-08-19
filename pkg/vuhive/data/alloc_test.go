package data_test

import (
	"fmt"
	"testing"

	"github.com/morphy76/vuhive/pkg/vuhive/data"
	"github.com/stretchr/testify/assert"
)

func TestAlloc_DataSet_Sequential(t *testing.T) {
	records := make([]data.Record, 100)
	for i := range 100 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Sequential)
	ctx := &mockContext{vuid: 1, iteration: 1}

	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = ds.Next(ctx)
	})

	assert.Equal(t, float64(0), allocs, "DataSet.Next (Sequential) must produce 0 heap allocations")
}

func TestAlloc_DataSet_Random(t *testing.T) {
	records := make([]data.Record, 100)
	for i := range 100 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.Random)
	ctx := &mockContext{vuid: 1, iteration: 1}

	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = ds.Next(ctx)
	})

	assert.Equal(t, float64(0), allocs, "DataSet.Next (Random) must produce 0 heap allocations")
}

func TestAlloc_DataSet_UniquePerVU(t *testing.T) {
	records := make([]data.Record, 100)
	for i := range 100 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.UniquePerVU)
	ctx := &mockContext{vuid: 1, iteration: 1}

	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = ds.Next(ctx)
	})

	assert.Equal(t, float64(0), allocs, "DataSet.Next (UniquePerVU) must produce 0 heap allocations")
}

func TestAlloc_DataSet_SharedQueue(t *testing.T) {
	records := make([]data.Record, 2000)
	for i := range 2000 {
		records[i] = data.Record{"id": fmt.Sprintf("%d", i)}
	}
	ds := data.NewDataSet(records, data.SharedQueue)
	ctx := &mockContext{vuid: 1, iteration: 1}

	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = ds.Next(ctx)
	})

	assert.Equal(t, float64(0), allocs, "DataSet.Next (SharedQueue) must produce 0 heap allocations")
}
