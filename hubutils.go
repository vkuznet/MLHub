package main

// client functions for ML backends
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/uptrace/bunrouter"
	"gopkg.in/mgo.v2/bson"
)

// Predict function fetches prediction for given uri, model and client's
// HTTP request. Code is based on the following example:
// https://golangbyexample.com/http-mutipart-form-body-golang/
func Predict(uri, model string, r *http.Request) ([]byte, error) {
	// parse incoming HTTP request multipart form
	err := r.ParseMultipartForm(32 << 20) // maxMemory

	// new multipart writer.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// create new field
	for k, vals := range r.MultipartForm.Value {
		for _, v := range vals {
			writer.WriteField(k, v)
		}
	}
	// add mandatory model field
	writer.WriteField("model", model)

	// parse and recreate file form
	for k, vals := range r.MultipartForm.File {
		for _, fh := range vals {
			fname := fh.Filename
			fw, err := writer.CreateFormFile(k, fname)
			if err != nil {
				log.Printf("ERROR: unable to create form file for key=%s fname=%s", k, fname)
				break
			}
			file, err := fh.Open()
			if err != nil {
				log.Printf("ERROR: unable to open fname=%s", fname)
				break
			}
			_, err = io.Copy(fw, file)
			if err != nil {
				log.Printf("ERROR: unable to copy fname=%s to multipart writer", fname)
				break
			}
		}
	}
	writer.Close()

	var data []byte
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	if Config.Verbose > 0 {
		log.Printf("POST request to %s with body\n%v", uri, string(body.Bytes()))
	}
	req, err := http.NewRequest("POST", uri, bytes.NewReader(body.Bytes()))
	if err != nil {
		return data, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rsp, err := client.Do(req)
	if rsp.StatusCode != http.StatusOK {
		log.Printf("Request failed with response code: %d", rsp.StatusCode)
	}
	defer rsp.Body.Close()
	data, err = io.ReadAll(rsp.Body)
	return data, err
}

func Download(model string) ([]byte, error) {
	// TODO: decide where we'll store ML models
	// either on disk or in MetaData database
	//     spec := bson.M{"model": model}
	//     records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	return []byte{}, nil
}

// Upload function uploads data for given model to MeataData database
func Upload(rec Record, r *http.Request) error {
	// read incoming data blog
	var data []byte
	var err error
	defer r.Body.Close()
	if r.Header.Get("Content-Encoding") == "gzip" {
		r.Header.Del("Content-Length")
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			return err
		}
		data, err = io.ReadAll(GzipReader{reader, r.Body})
	} else {
		data, err = io.ReadAll(r.Body)
	}
	if err != nil {
		return err
	}

	if _, ok := Config.MLBackends[rec.Type]; ok {
		// TODO: implement upload budle to MetaData database, and to specific ML backend
		// each backend may have different upload APIs
		log.Println("TODO: upload", string(data))
	}
	return nil
}

// helper function to get ML record for given HTTP request
func modelRecord(r *http.Request) (Record, error) {
	// look-up given ML name in MetaData database
	vars := mux.Vars(r)
	model, ok := vars["model"]
	if !ok { // no gorilla/mux, try bunrouter params map
		params := bunrouter.ParamsFromContext(r.Context())
		model, ok = params.Map()["model"]
	}

	var rec Record
	if ok {
		if Config.Verbose > 0 {
			log.Printf("get ML model %s meta-data", model)
		}
		// get ML meta-data
		spec := bson.M{"model": model}
		records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		if err != nil {
			msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
			return rec, errors.New(msg)
		}
		// we should have only one record from MetaData
		if len(records) != 1 {
			msg := fmt.Sprintf("Incorrect number of MetaData records %+v", records)
			return rec, errors.New(msg)
		}
		rec = records[0]
		return rec, nil
	}
	msg := fmt.Sprintf("unable to find %s model", model)
	err := errors.New(msg)
	return rec, err
}
