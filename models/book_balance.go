package models

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"general_ledger_golang/pkg/logger"
	"general_ledger_golang/pkg/util"
)

type BookBalance struct {
	Model
	BookId        string  `gorm:"primaryKey;index;column:bookId" json:"bookId"`
	AssetId       string  `gorm:"primaryKey;index;column:assetId" json:"assetId"`
	OperationType string  `gorm:"primaryKey;index;column:operationType" json:"operationType"`
	Balance       float64 `gorm:"type:numeric(32,8);check:non_negative_balance,balance >= 0 OR \"operationType\" != 'OVERALL' OR \"bookId\" = '1'" json:"balance"`
}

const (
	OverallOperation string = "OVERALL"
)

func (bB *BookBalance) ModifyBalance(operation map[string]interface{}, db *gorm.DB) error {
	log := logger.Logger.WithFields(logrus.Fields{
		"memo": operation["memo"],
		"op":   operation,
	})
	// commenting genericOp based validations to reduce updated rows on db
	// genericOp := util.DeepCopyMap(operation)

	overallOp := util.DeepCopyMap(operation)
	if _, ok := overallOp["metadata"]; !ok {
		return errors.New("metadata is not present, creation of book balance depends on metadata[\"operation\"], please send metadata with operation")
	}
	overallOp["metadata"].(map[string]interface{})["operation"] = OverallOperation

	operations := []map[string]interface{}{
		// genericOp,
		overallOp,
	}

	// sort the operation entries in place, everything is reference type, so sorting in-place works fine
	for _, op := range operations {
		entries := op["entries"].([]interface{})
		metadata := op["metadata"].(map[string]interface{})
		bB.sortEntries(entries)

		// create the queries by looping over the entries
		// note: Bulk upsert won't work here. for book_balance, there can be one bookId already present in book_balance
		// but the other one is not, so both will need to change. for one, it's insert and the other one it's update.
		queryList, params, err := GenerateUpsertCteQuery(entries, metadata)
		if err != nil {
			return err
		}

		log.Infof("Executing -> quries: %+v, params: %+v", queryList, params)

		// execute the queries one by one, if any query errors out, roll back
		for i, query := range queryList {
			t := db.Debug().Exec(query, params[i]...)
			if t.Error != nil {
				log.WithFields(map[string]interface{}{
					"q": map[string]interface{}{
						"query": strings.ReplaceAll(strings.ReplaceAll(query, "\t", " "), "\n", " "),
						"vars":  params[i],
					},
				}).Errorf("DB error, %+v", t.Error)

				return errors.New(t.Error.Error())
			}
		}
	}

	return nil
}

// GenerateBulkUpsertQuery will generate a single bulkUpsert query
func GenerateBulkUpsertQuery(entries []interface{}, metadata map[string]interface{}) (query string, params []interface{}, errs error) {
	var bookIds []string
	var assetIds []string
	var operationTypes []string
	var values []string

	bulkInsertQ := `INSERT INTO book_balances 
				(
					"assetId", 
					"bookId", 
					"operationType", 
					"balance", 
					"createdAt", 
					"updatedAt" 
				) 
			SELECT 
				unnest(string_to_array(?, ',')),
				unnest(string_to_array(?, ',')),
				unnest(string_to_array(?, ',')),
				unnest(string_to_array(?, ',')::numeric[]),
				NOW()::timestamp, 
				NOW()::timestamp `
	updateQ := `UPDATE book_balances 
					SET balance = book_balances.balance + data_table.value::numeric, 
					"updatedAt" = NOW()::timestamp 
				FROM 
					( 
						SELECT unnest(string_to_array(?, ',')) as "assetId",
								unnest(string_to_array(?, ',')) as "bookId", 
								unnest(string_to_array(?, ',')) as "operationType", 
								unnest(string_to_array(?, ',')::numeric[]) as value 
					) as data_table 
				WHERE
					book_balances."assetId" = data_table."assetId" 
					AND book_balances."bookId" = data_table."bookId" 
					AND book_balances."operationType" = data_table."operationType" 
				RETURNING *`

	cteQ := fmt.Sprintf(`
			WITH upsert AS (
                        %s
			)
			%s
			WHERE NOT EXISTS(
				SELECT * FROM upsert
			);
			`, updateQ, bulkInsertQ)
	var errList []string
	for _, entry2 := range entries {
		entry := entry2.(map[string]interface{})

		if util.Includes(entry["bookId"], []interface{}{1, -1, "1", "-1"}) {
			continue
		}

		operationType := metadata["operation"]

		if operationType == nil {
			return "", nil, errors.New("operation is not present inside metadata, creation of book balance depends on metadata[\"operation\"], please send metadata with operation")
		}
		//valueAsF64, parseFloatErr := strconv.ParseFloat(entry["value"].(string), 64)
		//
		//if parseFloatErr != nil {
		//	errList = append(errList, parseFloatErr.Error())
		//}

		bookIds = append(bookIds, entry["bookId"].(string))
		assetIds = append(assetIds, entry["assetId"].(string))
		values = append(values, entry["value"].(string))
		operationTypes = append(operationTypes, operationType.(string))
	}
	if len(errList) > 0 {
		return "", nil, errors.New(strings.Join(errList, ""))
	}
	params = append(params,
		strings.Join(assetIds, ","),
		strings.Join(bookIds, ","),
		strings.Join(operationTypes, ","),
		strings.Join(values, ","),
		strings.Join(assetIds, ","),
		strings.Join(bookIds, ","),
		strings.Join(operationTypes, ","),
		strings.Join(values, ","),
	)
	return cteQ, params, nil
}

// GenerateUpsertCteQuery will generate multiple upsert queries
func GenerateUpsertCteQuery(entries []interface{}, metadata map[string]interface{}) (queryList []string, params [][]interface{}, err error) {
	for _, entry2 := range entries {
		entry := entry2.(map[string]interface{})
		var paramsSlice []interface{}
		// Uses environment variable to decide which accounts should be tracked inside the book balance table.
		// EXCLUDED_BALANCE_BOOK_IDS if not provided, will store every bookId in the balances table.
		if strings.Contains(os.Getenv("EXCLUDED_BALANCE_BOOK_IDS"), entry["bookId"].(string)) {
			continue
		}
		operationType := metadata["operation"]

		if operationType == nil {
			return nil, nil, errors.New("operation is not present inside metadata, creation of book balance depends on metadata[\"operation\"], please send metadata with operation")
		}

		updateQ := `
				UPDATE book_balances
					SET
					balance = ?,
					"updatedAt" = NOW()::timestamp
				WHERE
					"assetId" = ?
					AND "bookId" = ?
					AND "operationType" = ?
				RETURNING *
			`
		paramsSlice = append(paramsSlice, gorm.Expr("book_balances.balance + ?::numeric ", entry["value"]), entry["assetId"], entry["bookId"], operationType)

		insertQ := `INSERT
			INTO book_balances
			(
				"bookId",
				"assetId",
				"operationType",
				balance,
				"createdAt",
				"updatedAt"
			)
			SELECT 
				?,
				?,
				?,
				?,
				NOW()::timestamp, 
				NOW()::timestamp`

		paramsSlice = append(
			paramsSlice,
			entry["bookId"],
			entry["assetId"],
			operationType,
			entry["value"])

		cteQ := fmt.Sprintf(`
			WITH upsert AS (
                        %s
                    	)
			%s
			WHERE NOT EXISTS(
				SELECT * FROM upsert
			);
			`, updateQ, insertQ)

		queryList = append(queryList, cteQ)
		params = append(params, paramsSlice)
	}
	return queryList, params, nil
}

func (bB *BookBalance) sortEntries(entries []interface{}) {
	sort.SliceStable(entries, func(i, j int) bool {
		iEntry := entries[i].(map[string]interface{})
		jEntry := entries[j].(map[string]interface{})
		// i < j means smallest first, largest last
		sortByBookId := iEntry["bookId"].(string) < jEntry["bookId"].(string)
		// first sort by bookId, if bookIds are matching, then sort by assetId, to guarantee order.
		if iEntry["bookId"] == jEntry["bookId"] {
			sortByAssetId := iEntry["assetId"].(string) < jEntry["assetId"].(string)
			return sortByAssetId
		}
		return sortByBookId
	})
}

// GetBalance fetches the balance
func (bB *BookBalance) GetBalance(bookId, assetId, operationType string, tx *gorm.DB) (*[]BookBalance, error) {
	var d *gorm.DB
	if bookId == "" {
		return nil, errors.New("BookId is missing")
	}
	if tx != nil {
		d = tx
	} else {
		d = db
	}
	var balance []BookBalance
	//err := db.Select("id").Where(Auth{Username: username, Password: password}).First(&auth).Error
	query := d.Where(BookBalance{BookId: bookId})

	if assetId != "" {
		query.Where(BookBalance{AssetId: assetId})
	}

	if operationType != "" {
		query.Where(BookBalance{OperationType: operationType})
	} else {
		query.Where(BookBalance{OperationType: OverallOperation})
	}

	t := query.Select("bookId", "assetId", "balance", "operationType").Find(&balance)

	if t.RowsAffected < 1 {
		return nil, nil
	}

	return &balance, nil
}
