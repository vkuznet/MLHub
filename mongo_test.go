package main

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// TestMongoInsert
func TestMongoInsert(t *testing.T) {
	initMetaDataService()

	// our db attributes
	dbname := "ml"
	collname := "metadata"

	// remove all records in test collection
	MongoRemove(dbname, collname, bson.M{})

	// insert one record
	var records []Record
	var err error
	rec := Record{
		Model:       "model",
		Type:        "type",
		Version:     "version",
		Description: "description",
		Reference:   "reference",
		Discipline:  "domains",
		Bundle:      "bundle",
		UserName:    "user",
		UserID:      "id",
		Provider:    "provider",
	}
	records = append(records, rec)
	MongoInsert(dbname, collname, records)

	// look-up one record
	spec := bson.M{"model": "model"}
	idx := 0
	limit := 1
	records, err = MongoGet(dbname, collname, spec, idx, limit)
	if err != nil {
		t.Errorf("unable to find records using spec '%s', error %v", spec, err)
	}
	if len(records) != 1 {
		t.Errorf("wrong number of records using spec '%s', records %+v", spec, records)
	}
}
