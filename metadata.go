package main

import (
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

// Record define ML mongo record
type Record struct {
	MetaData    map[string]interface{} `json:"meta_data"`   // meta-data information about ML model
	Model       string                 `json:"model"`       // model name
	Type        string                 `json:"type"`        // model type
	Version     string                 `json:"version"`     // ML version
	Description string                 `json:"description"` // ML model description
	Reference   string                 `json:"reference"`   // ML reference URL
	Discipline  string                 `json:"discipline"`  // ML discipline
	Bundle      string                 `json:"bundle"`      // ML bundle file
}

// ToJSON provides string representation of Record
func (r Record) ToJSON() string {
	// create pretty JSON representation of the record
	data, _ := json.MarshalIndent(r, "", "    ")
	return string(data)
}

// MetaData represents meta-data database object
type MetaData struct {
	DBName string
	DBColl string
}

// Insert inserts record into MetaData database
func (m *MetaData) Insert(rec Record) error {
	records := []Record{rec}
	err := MongoUpsert(Config.DBName, Config.DBColl, records)
	return err
}

// Update updates record in MetaData database
func (m *MetaData) Update(rec Record) error {
	spec := bson.M{"model": rec.Model}
	meta := bson.M{"model": rec.Model, "type": rec.Type, "meta_data": rec.MetaData}
	err := MongoUpdate(Config.DBName, Config.DBColl, spec, meta)
	return err
}

// Remove removes given model from MetaData database
func (m *MetaData) Remove(model string) error {
	spec := bson.M{"name": model}
	err := MongoRemove(Config.DBName, Config.DBColl, spec)
	return err
}

// Records retrieves records from underlying MetaData database
func (m *MetaData) Records(model, mlType, version string) ([]Record, error) {
	spec := bson.M{}
	if model != "" {
		spec["model"] = model
	}
	if version != "" {
		spec["version"] = version
	}
	if mlType != "" {
		spec["type"] = mlType
	}
	records, err := MongoGet(m.DBName, m.DBColl, spec, 0, -1)
	return records, err
}
