package main

// handlers module holds all HTTP handlers functions
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"fmt"
	"net/http"
)

// FaviconHandler
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/images/favicon.ico", Config.StaticDir))
}
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement delete API from all backend servers
	RequestHandler(w, r)
}
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement upload API from all backend servers
	RequestHandler(w, r)
}
func PredictHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement predict API from all backend servers
	RequestHandler(w, r)
}
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement models API from all backend servers
	RequestHandler(w, r)
}
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement status API from all backend servers
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	// redirect HTTP requests based on provided request
	// TODO: we need to analyze incoming HTTP request to determine
	// which backend URL to use
	backendURL := "http://localhost:8083"
	reverseProxy(backendURL, w, r)
}
