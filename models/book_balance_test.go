package models

import (
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	// GenerateUpsertCteQuery will generate multiple upsert queries

	queries, params, err := GenerateUpsertCteQuery([]interface{}{
		map[string]interface{}{
			"assetId": "btc",
			"bookId":  "3",
			"value":   "-1",
		},
		map[string]interface{}{
			"assetId": "btc",
			"bookId":  "4",
			"value":   "1",
		},
	}, map[string]interface{}{
		"operation": "BLOCK",
	})

	if len(queries) != 2 {
		t.Fatalf("Queries should have 2 elements")
	}

	if len(params) != 2 {
		t.Fatalf("Params should have 2 elements")
	}

	if err != nil {
		t.Fatalf("Err should be nil")
	}

	t.Log(strings.Join(queries, "\n"), params, err)
}
