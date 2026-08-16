package data

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

// Record represents a single data record mapping string keys to string values.
type Record = map[string]string

// DataSet manages a collection of records and their distribution strategy across VUs.
type DataSet struct {
	records  []Record
	strategy Strategy
	cursor   int64
}

// NewDataSet creates a DataSet with the given records and distribution strategy.
func NewDataSet(records []Record, strategy Strategy) *DataSet {
	return &DataSet{
		records:  records,
		strategy: strategy,
	}
}

// Len returns the total number of records in the dataset.
func (ds *DataSet) Len() int {
	return len(ds.records)
}

// Records returns the underlying slice of records.
func (ds *DataSet) Records() []Record {
	return ds.records
}

// Next selects the next record based on the dataset's strategy and the given context.
func (ds *DataSet) Next(ctx ContextAccessor) (Record, error) {
	if len(ds.records) == 0 {
		return nil, ErrDatasetExhausted
	}

	n := int64(len(ds.records))

	switch ds.strategy {
	case Sequential, UniquePerVU:
		if ctx == nil {
			return nil, ErrNilContext
		}
		vuid := ctx.VUID()
		iteration := ctx.Iteration()
		vuidOffset := vuid - 1
		if vuidOffset < 0 {
			vuidOffset = 0
		}
		idx := (vuidOffset + iteration) % n
		return ds.records[idx], nil

	case Random:
		idx := rand.Int64N(n)
		return ds.records[idx], nil

	case SharedQueue:
		idx := atomic.AddInt64(&ds.cursor, 1) - 1
		if idx >= n {
			return nil, ErrDatasetExhausted
		}
		return ds.records[idx], nil

	default:
		if ctx == nil {
			return nil, ErrNilContext
		}
		idx := ctx.Iteration() % n
		return ds.records[idx], nil
	}
}

// LoadCSV parses CSV data from r (using the first row as header titles) into a DataSet.
func LoadCSV(r io.Reader, strategy Strategy) (*DataSet, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("vuhive/data: failed to read CSV header: %w", err)
	}

	var records []Record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("vuhive/data: CSV parse error: %w", err)
		}

		rec := make(Record, len(headers))
		for i, h := range headers {
			if i < len(row) {
				rec[h] = row[i]
			} else {
				rec[h] = ""
			}
		}
		records = append(records, rec)
	}

	return NewDataSet(records, strategy), nil
}

// LoadJSON parses a JSON array of objects from r into a DataSet.
func LoadJSON(r io.Reader, strategy Strategy) (*DataSet, error) {
	var rawItems []map[string]any
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&rawItems); err != nil {
		return nil, fmt.Errorf("vuhive/data: JSON parse error: %w", err)
	}

	var records []Record
	for _, item := range rawItems {
		records = append(records, parseObjectToRecord(item))
	}

	return NewDataSet(records, strategy), nil
}

// LoadJSONL parses newline-delimited JSON objects from r into a DataSet.
func LoadJSONL(r io.Reader, strategy Strategy) (*DataSet, error) {
	scanner := bufio.NewScanner(r)
	var records []Record

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item map[string]any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("vuhive/data: JSONL line parse error: %w", err)
		}
		records = append(records, parseObjectToRecord(item))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("vuhive/data: JSONL scan error: %w", err)
	}

	return NewDataSet(records, strategy), nil
}

// LoadCSVFile opens the specified file path and parses CSV data into a DataSet.
func LoadCSVFile(filePath string, strategy Strategy) (*DataSet, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("vuhive/data: failed to open CSV file %q: %w", filePath, err)
	}
	defer func() {
		_ = f.Close()
	}()
	return LoadCSV(f, strategy)
}

// LoadJSONFile opens the specified file path and parses JSON data into a DataSet.
func LoadJSONFile(filePath string, strategy Strategy) (*DataSet, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("vuhive/data: failed to open JSON file %q: %w", filePath, err)
	}
	defer func() {
		_ = f.Close()
	}()
	return LoadJSON(f, strategy)
}

// LoadJSONLFile opens the specified file path and parses JSONL data into a DataSet.
func LoadJSONLFile(filePath string, strategy Strategy) (*DataSet, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("vuhive/data: failed to open JSONL file %q: %w", filePath, err)
	}
	defer func() {
		_ = f.Close()
	}()
	return LoadJSONL(f, strategy)
}

func parseObjectToRecord(item map[string]any) Record {
	rec := make(Record, len(item))
	for k, v := range item {
		rec[k] = stringifyValue(v)
	}
	return rec
}

func stringifyValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(val, 10)
	case int:
		return strconv.Itoa(val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}
