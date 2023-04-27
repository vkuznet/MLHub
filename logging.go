package main

// logging module provides various logging methods
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// LogRecord represents data we can send to StompAMQ or HTTP endpoint
type LogRecord struct {
	Method         string  `json:"method"`           // http.Request HTTP method
	URI            string  `json:"uri"`              // http.RequestURI
	API            string  `json:"api"`              // http service API being used
	System         string  `json:"system"`           // cmsweb service name
	ClientIP       string  `json:"clientip"`         // client IP address
	BytesIn        int64   `json:"bytes_in"`         // number of bytes send with HTTP request
	BytesOut       int64   `json:"bytes_out"`        // number of bytes received with HTTP request
	Proto          string  `json:"proto"`            // http.Request protocol
	Status         int64   `json:"status"`           // http.Request status code
	ContentLength  int64   `json:"content_length"`   // http.Request content-length
	Referer        string  `json:"referer"`          // http referer
	UserAgent      string  `json:"user_agent"`       // http user-agent field
	XForwardedHost string  `json:"x_forwarded_host"` // http.Request X-Forwarded-Host
	XForwardedFor  string  `json:"x_forwarded_for"`  // http.Request X-Forwarded-For
	RemoteAddr     string  `json:"remote_addr"`      // http.Request remote address
	RequestTime    float64 `json:"request_time"`     // http request time
	Timestamp      int64   `json:"timestamp"`        // record timestamp
}

// helper function to produce UTC time prefixed output
func utcMsg(data []byte) string {
	s := string(data)
	v, e := url.QueryUnescape(s)
	if e == nil {
		return v
	}
	return s
}

// custom rotate logger
type rotateLogWriter struct {
	RotateLogs *rotatelogs.RotateLogs
}

func (w rotateLogWriter) Write(data []byte) (int, error) {
	return w.RotateLogs.Write([]byte(utcMsg(data)))
}

// custom logger
type logWriter struct {
}

func (writer logWriter) Write(data []byte) (int, error) {
	return fmt.Print(utcMsg(data))
}

// helper function to log every single user request, here we pass pointer to status code
// as it may change through the handler while we use defer logRequest
func logRequest(w http.ResponseWriter, r *http.Request, start time.Time, status int, tstamp int64, bytesOut int64) {
	dataMsg := fmt.Sprintf("[data: %v in %v out]", r.ContentLength, bytesOut)
	referer := r.Referer()
	if referer == "" {
		referer = "-"
	}
	//     var clientip string
	//     xff := r.Header.Get("X-Forwarded-For")
	//     if xff != "" {
	//         clientip = strings.Split(xff, ":")[0]
	//     } else if r.RemoteAddr != "" {
	//         clientip = strings.Split(r.RemoteAddr, ":")[0]
	//     }
	addr := r.RemoteAddr
	refMsg := fmt.Sprintf("[ref: \"%s\" \"%v\"]", referer, r.Header.Get("User-Agent"))
	respMsg := fmt.Sprintf("[req: %v]", time.Since(start))
	uri, err := url.QueryUnescape(r.RequestURI)
	if err != nil {
		log.Println("unable to unescape request uri", err)
		uri = r.RequestURI
	}
	t := time.Now().Format(time.RFC3339)
	log.Printf("%s %s %d %s %s %s %s %s %s\n", t, r.Proto, status, addr, r.Method, uri, dataMsg, refMsg, respMsg)
	/*
		rec := LogRecord{
			Method:         r.Method,
			URI:            r.RequestURI,
			API:            getAPI(r.RequestURI),
			BytesIn:        r.ContentLength,
			BytesOut:       bytesOut,
			Proto:          r.Proto,
			Status:         int64(status),
			ContentLength:  r.ContentLength,
			Referer:        referer,
			UserAgent:      r.Header.Get("User-Agent"),
			XForwardedHost: r.Header.Get("X-Forwarded-Host"),
			XForwardedFor:  xff,
			ClientIP:       clientip,
			RemoteAddr:     r.RemoteAddr,
			RequestTime:    time.Since(start).Seconds(),
			Timestamp:      tstamp,
		}
	*/
}

// helper function to extract service API from the record URI
func getAPI(uri string) string {
	// /httpgo?test=bla
	arr := strings.Split(uri, "/")
	last := arr[len(arr)-1]
	arr = strings.Split(last, "?")
	return arr[0]
}
