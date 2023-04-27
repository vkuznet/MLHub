package main

// mongo module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet AT gmail dot com>
//
// References : https://gist.github.com/boj/5412538
//              https://gist.github.com/border/3489566

import (
	"encoding/json"
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Record define ML mongo record
type Record struct {
	Meta map[string]interface{} // meta-data information about ML model
	Name string                 // model name
	Type string                 // model type
}

// ToJSON provides string representation of Record
func (r Record) ToJSON() string {
	// create pretty JSON representation of the record
	data, _ := json.MarshalIndent(r, "", "    ")
	return string(data)
}

// MongoConnection defines connection to MongoDB
type MongoConnection struct {
	Session *mgo.Session
}

// Connect provides connection to MongoDB
func (m *MongoConnection) Connect() (*mgo.Session, error) {
	var err error
	if m.Session == nil {
		m.Session, err = mgo.Dial(Config.DBURI)
		if err != nil {
			return nil, err
		}
		//         m.Session.SetMode(mgo.Monotonic, true)
		m.Session.SetMode(mgo.Strong, true)
	}
	return m.Session.Clone(), nil
}

// global object which holds MongoDB connection
var _Mongo MongoConnection

// MongoInsert records into MongoDB
func MongoInsert(dbname, collname string, records []Record) {
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	for _, rec := range records {
		if err := c.Insert(&rec); err != nil {
			log.Printf("Fail to insert record %v, error %v\n", rec, err)
		}
	}
}

// MongoUpsert records into MongoDB
func MongoUpsert(dbname, collname string, records []Record) error {
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return err
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	for _, rec := range records {
		model := rec.Name
		if model == "" {
			log.Printf("no model, record %v\n", rec)
			continue
		}
		spec := bson.M{"model": model}
		if _, err := c.Upsert(spec, &rec); err != nil {
			log.Printf("Fail to insert record %v, error %v\n", rec, err)
			return err
		}
	}
	return nil
}

// MongoGet records from MongoDB
func MongoGet(dbname, collname string, spec bson.M, idx, limit int) []Record {
	out := []Record{}
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return out
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	if limit > 0 {
		err = c.Find(spec).Skip(idx).Limit(limit).All(&out)
	} else {
		err = c.Find(spec).Skip(idx).All(&out)
	}
	if err != nil {
		log.Printf("Unable to get records, error %v\n", err)
	}
	return out
}

// MongoGetSorted records from MongoDB sorted by given key
func MongoGetSorted(dbname, collname string, spec bson.M, skeys []string) []Record {
	out := []Record{}
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return out
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	err = c.Find(spec).Sort(skeys...).All(&out)
	if err != nil {
		log.Printf("Unable to sort records, error %v\n", err)
		// try to fetch all unsorted data
		err = c.Find(spec).All(&out)
		if err != nil {
			log.Printf("Unable to find records, error %v\n", err)
		}
	}
	return out
}

// helper function to present in bson selected fields
func sel(q ...string) (r bson.M) {
	r = make(bson.M, len(q))
	for _, s := range q {
		r[s] = 1
	}
	return
}

// MongoUpdate inplace for given spec
func MongoUpdate(dbname, collname string, spec, newdata bson.M) {
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	err = c.Update(spec, newdata)
	if err != nil {
		log.Printf("Unable to update record, spec %v, data %v, error %v\n", spec, newdata, err)
	}
}

// MongoCount gets number records from MongoDB
func MongoCount(dbname, collname string, spec bson.M) int {
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return 0
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	nrec, err := c.Find(spec).Count()
	if err != nil {
		log.Printf("Unable to count records, spec %v, error %v\n", spec, err)
	}
	return nrec
}

// MongoRemove records from MongoDB
func MongoRemove(dbname, collname string, spec bson.M) {
	s, err := _Mongo.Connect()
	if err != nil {
		log.Println("Unable to connect to MongoDB", err)
		return
	}
	defer s.Close()
	c := s.DB(dbname).C(collname)
	_, err = c.RemoveAll(spec)
	if err != nil && err != mgo.ErrNotFound {
		log.Printf("Unable to remove records, spec %v, error %v\n", spec, err)
	}
}
