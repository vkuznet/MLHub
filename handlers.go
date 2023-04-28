package main

// handlers module holds all HTTP handlers functions
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

// FaviconHandler
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/images/favicon.ico", Config.StaticDir))
}

// GetHandler handles GET HTTP requests
func GetHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: I need to manage how to redirect requests to backend
	// ML server APIs, so far redirect to TFaaS
	backendURL := "http://localhost:8083"
	reverseProxy(backendURL, w, r)
}

// PostHandler handles POST HTTP requests,
// this request will create and upload ML models to backend server(s)
func PostHandler(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// - create new entry in MetaData database
	// - call backend API to create upload new ML tarball
	RequestHandler(w, r)
}

// PutHandler handles PUT HTTP requests, this request will
// update ML model in backend or MetaData database
func PutHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement upload API from all backend servers
	RequestHandler(w, r)
}

// GetHandler handles GET HTTP requests, this request will
// delete ML model in backend and MetaData database
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if model, ok := vars["model"]; ok {
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
	records := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	data, err := json.Marshal(records)
	if err != nil {
		msg := fmt.Sprintf("unable to marshal data, error=%v", err)
		httpError(w, r, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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
