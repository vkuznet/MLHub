package main

// handlers module holds all HTTP handlers functions
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
)

// HTTPError represents HTTP error record
type HTTPError struct {
	Method         string `json:"method"`           // HTTP method
	HTTPCode       int    `json:"http_code"`        // HTTP error code
	Code           int    `json:"code"`             // server status code
	Timestamp      string `json:"timestamp"`        // timestamp of the error
	Path           string `json:"path"`             // URL path
	UserAgent      string `json:"user_agent"`       // http user-agent field
	XForwardedHost string `json:"x_forwarded_host"` // http.Request X-Forwarded-Host
	XForwardedFor  string `json:"x_forwarded_for"`  // http.Request X-Forwarded-For
	RemoteAddr     string `json:"remote_addr"`      // http.Request remote address
	Reason         string `json:"reason"`           // error message
}

// helper function to provide standard HTTP error reply
func httpError(w http.ResponseWriter, r *http.Request, code int, err error, httpCode int) {
	hrec := HTTPError{
		Method:         r.Method,
		Timestamp:      time.Now().String(),
		Path:           r.RequestURI,
		RemoteAddr:     r.RemoteAddr,
		XForwardedFor:  r.Header.Get("X-Forwarded-For"),
		XForwardedHost: r.Header.Get("X-Forwarded-Host"),
		UserAgent:      r.Header.Get("User-agent"),
		Code:           code,
		Reason:         err.Error(),
		HTTPCode:       httpCode,
	}
	if data, err := json.Marshal(hrec); err == nil {
		w.WriteHeader(httpCode)
		w.Write(data)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func checkRecord(rec Record, model string) error {
	if rec.Model != model {
		err := errors.New(fmt.Sprintf("reqested ML model %s is not equal to meta-data model name %s", model, rec.Model))
		return err
	}
	if rec.Type == "" {
		err := errors.New(fmt.Sprintf("ML type is missing, please provide one of %+v", MLTypes))
		return err
	}
	if !InList(rec.Type, MLTypes) {
		err := errors.New(fmt.Sprintf("ML type %s is not supported, please provide one of %+v", rec.Type, MLTypes))
		return err
	}
	if rec.MetaData == nil {
		err := errors.New(fmt.Sprintf("Missing meta_data"))
		return err
	}
	return nil
}

// helper function to check if HTTP request contains form-data
func formData(r *http.Request) bool {
	for key, values := range r.Header {
		if strings.ToLower(key) == "content-type" {
			for _, v := range values {
				if strings.Contains(strings.ToLower(v), "form-data") {
					return true
				}
			}
		}
	}
	return false
}

// FaviconHandler
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/images/favicon.ico", Config.StaticDir))
}

// PredictHandler handles GET HTTP requests
func PredictHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("request predictions from ML model %s", model)
		}
		// get ML meta-data
		spec := bson.M{"model": model}
		records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		if err != nil {
			msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
			httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		// we should have only one record from MetaData
		if len(records) != 1 {
			msg := fmt.Sprintf("Incorrect number of MetaData records %+v", records)
			httpError(w, r, MetaDataError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		rec := records[0]
		if Config.Verbose > 0 {
			log.Printf("use ML MetaData record %+v", rec)
		}
		if backend, ok := Config.MLBackends[rec.Type]; ok {
			path := r.RequestURI
			bPath := strings.Replace(path, fmt.Sprintf("/model/%s", model), "", -1)
			uri := fmt.Sprintf("%s", backend.URI)
			rurl := uri + bPath
			if Config.Verbose > 0 {
				log.Printf("get predictions from %s model at %s", model, rurl)
			}
			data, err := Predict(rurl, model, r)
			if err == nil {
				w.Write(data)
			} else {
				httpError(w, r, BadRequest, err, http.StatusBadRequest)
			}
			//             reverseProxy(uri, w, r)
		}
		return
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// DownloadHandler handles download action of ML model from back-end server
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	// look-up given ML name in MetaData database
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("get ML model %s meta-data", model)
		}
		// get ML meta-data
		spec := bson.M{"model": model}
		records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		if err != nil {
			msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
			httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		// we should have only one record from MetaData
		if len(records) != 1 {
			msg := fmt.Sprintf("Incorrect number of MetaData records %+v", records)
			httpError(w, r, MetaDataError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		rec := records[0]
		if backend, ok := Config.MLBackends[rec.Type]; ok {
			bundle, err := backend.Download(rec.Model)
			if err != nil {
				httpError(w, r, BadRequest, errors.New("unable to download data from backend"), http.StatusInternalServerError)
				return
			}
			w.Write(bundle)
		}
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// UploadHandler handles upload action of ML model to back-end server
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// look-up given ML name in MetaData database
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("get ML model %s meta-data", model)
		}
		// get ML meta-data
		spec := bson.M{"model": model}
		records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		if err != nil {
			msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
			httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		// we should have only one record from MetaData
		if len(records) != 1 {
			msg := fmt.Sprintf("Incorrect number of MetaData records %+v", records)
			httpError(w, r, MetaDataError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		rec := records[0]
		// check if we provided with proper form data
		if !formData(r) {
			httpError(w, r, BadRequest, errors.New("unable to get form data"), http.StatusBadRequest)
			return
		}
		// read incoming data blog
		var data []byte
		defer r.Body.Close()
		if r.Header.Get("Content-Encoding") == "gzip" {
			r.Header.Del("Content-Length")
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				httpError(w, r, BadRequest, errors.New("unable to get gzip reader"), http.StatusInternalServerError)
				return
			}
			data, err = io.ReadAll(GzipReader{reader, r.Body})
		} else {
			data, err = io.ReadAll(r.Body)
		}
		if err != nil {
			httpError(w, r, BadRequest, errors.New("unable to read body"), http.StatusBadRequest)
			return
		}

		if backend, ok := Config.MLBackends[rec.Type]; ok {
			if err := backend.Upload(data); err != nil {
				httpError(w, r, BadRequest, errors.New("unable to upload data to backend"), http.StatusInternalServerError)
				return
			}
		}
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// GetHandler handles GET HTTP requests
func GetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("get ML model %s meta-data", model)
		}
		// get ML meta-data
		spec := bson.M{"model": model}
		records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		if err != nil {
			msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
			httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(records)
		if err != nil {
			msg := fmt.Sprintf("unable to marshal data, error=%v", err)
			httpError(w, r, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// PostHandler handles POST HTTP requests,
// this request will create and upload ML models to backend server(s)
func PostHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add code to create ML model on backend
	// so far the code below only creates ML model info in MetaData database
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("update ML model %s", model)
		}
		// parse input JSON body
		decoder := json.NewDecoder(r.Body)
		var rec Record
		err := decoder.Decode(&rec)
		if err != nil {
			httpError(w, r, MetaDataRecordError, err, http.StatusBadRequest)
			return
		}
		if err := checkRecord(rec, model); err != nil {
			httpError(w, r, MetaDataRecordError, err, http.StatusBadRequest)
			return
		}
		// update ML meta-data
		records := []Record{rec}
		err = MongoUpsert(Config.DBName, Config.DBColl, records)
		if err != nil {
			httpError(w, r, DatabaseError, err, http.StatusInternalServerError)
		}
		return
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// PutHandler handles PUT HTTP requests, this request will
// update ML model in backend or MetaData database
func PutHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add code to update ML model on backend
	// so far the code below only updates ML model in MetaData database
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("update ML model %s", model)
		}
		// parse input JSON body
		decoder := json.NewDecoder(r.Body)
		var rec Record
		err := decoder.Decode(&rec)
		if err != nil {
			httpError(w, r, MetaDataRecordError, err, http.StatusBadRequest)
			return
		}
		if err := checkRecord(rec, model); err != nil {
			httpError(w, r, MetaDataRecordError, err, http.StatusBadRequest)
			return
		}
		// update ML meta-data
		spec := bson.M{"model": model}
		meta := bson.M{"model": model, "type": rec.Type, "meta_data": rec.MetaData}
		err = MongoUpdate(Config.DBName, Config.DBColl, spec, meta)
		if err != nil {
			httpError(w, r, DatabaseError, err, http.StatusInternalServerError)
		}
		return
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// GetHandler handles GET HTTP requests, this request will
// delete ML model in backend and MetaData database
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
		if Config.Verbose > 0 {
			log.Printf("delete ML model %s", model)
		}
		// delete ML model in MetaData database
		spec := bson.M{"name": model}
		err := MongoRemove(Config.DBName, Config.DBColl, spec)
		if err != nil {
			httpError(w, r, DatabaseError, err, http.StatusInternalServerError)
		}
		return
	}
	httpError(w, r, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// ModelsHandler provides information about registered ML models
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	spec := bson.M{}
	records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	if err != nil {
		msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
		httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(records)
	if err != nil {
		msg := fmt.Sprintf("unable to marshal data, error=%v", err)
		httpError(w, r, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
		return
	}
	w.Write(data)
	return
}

// StatusHandler handles status of MLHub server
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement status API from all backend servers
}

// RequestHandler handles incoming HTTP requests
func RequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		GetHandler(w, r)
	} else if r.Method == "POST" {
		PostHandler(w, r)
	} else if r.Method == "PUT" {
		PutHandler(w, r)
	} else if r.Method == "DELETE" {
		DeleteHandler(w, r)
	} else {
		msg := fmt.Sprintf("Unsupport HTTP method %s", r.Method)
		httpError(w, r, BadRequest, errors.New(msg), http.StatusInternalServerError)
	}
}
