// Package data provides test data parameterization, dataset loaders, and distribution strategies.
//
// Supported loaders include CSV (with headers), JSON arrays, and JSONL (newline-delimited JSON)
// from io.Reader streams or filesystem paths.
//
// Supported distribution strategies across Virtual Users and iterations:
//   - Sequential: Round-robin across records deterministically by VU ID and iteration.
//   - Random: Thread-safe uniform random selection across records.
//   - UniquePerVU: Deterministic partition/offset of records per Virtual User.
//   - SharedQueue: Thread-safe single-pass queue dispensing each record exactly once until exhausted.
//
// Basic Usage:
//
//	ds, err := data.LoadCSVFile("users.csv", data.Sequential)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Inside RunVU:
//	record, err := ds.Next(ctx)
//	if err != nil {
//		return err
//	}
//	username := record["username"]
package data
