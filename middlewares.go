package main

// middleware module provides various middleware modules for proxy server
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	limiter "github.com/ulule/limiter/v3"
	stdlib "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	memory "github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/uptrace/bunrouter"
)

// limiter middleware pointer
var limiterMiddleware *stdlib.Middleware

// initialize Limiter middleware pointer
func initLimiter(period string) {
	log.Printf("limiter rate='%s'", period)
	// create rate limiter with 5 req/second
	rate, err := limiter.NewRateFromFormatted(period)
	if err != nil {
		panic(err)
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	limiterMiddleware = stdlib.NewMiddleware(instance)
}

// Validate should implement input validation
func Validate(r *http.Request) error {
	return nil
}

// mux (http.Handler) validate middleware to validate incoming requests' parameters
func validateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			next.ServeHTTP(w, r)
			return
		}
		// perform validation of input parameters
		err := Validate(r)
		if err != nil {
			uri, _ := url.QueryUnescape(r.RequestURI)
			log.Printf("HTTP %s %s validation error %v\n", r.Method, uri, err)
			w.WriteHeader(http.StatusBadRequest)
			rec := make(map[string]string)
			rec["error"] = fmt.Sprintf("Validation error %v", err)
			if r, e := json.Marshal(rec); e == nil {
				w.Write(r)
			}
			return
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// mux (http.Handler) limitier middleware to limit incoming requests
func limitMiddleware(next http.Handler) http.Handler {
	return limiterMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}))
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// wrapper for response writer
// based on https://blog.questionable.services/article/guide-logging-middleware-go/
func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true

	return
}

// mux (http.Handler) logging middleware to log the incoming HTTP request and its duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		tstamp := int64(start.UnixNano() / 1000000) // use milliseconds for MONIT

		wrapped := wrapResponseWriter(w)
		status := wrapped.status
		if status == 0 { // the status code was not set, i.e. everything is fine
			status = 200
		}
		next.ServeHTTP(wrapped, r)
		var dataSize int64
		logRequest(w, r, start, wrapped.status, tstamp, dataSize)
	})
}

/*
 * bunrouter middlewares based on bunrouter.HandlerFunc (http.HandlerFunc)
 */

// bunrouer logging middelware implementation
func bunrouterLoggingMiddleware(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, r bunrouter.Request) error {
		start := time.Now()
		tstamp := int64(start.UnixNano() / 1000000) // use milliseconds for MONIT

		wrapped := wrapResponseWriter(w)
		status := wrapped.status
		if status == 0 { // the status code was not set, i.e. everything is fine
			status = 200
		}
		if err := next(wrapped, r); err != nil {
			return err
		}
		var dataSize int64
		logRequest(w, r.Request, start, wrapped.status, tstamp, dataSize)
		return nil
	}
}

// bunrouter limiter middleware implementation, based on
// https://github.com/ulule/limiter/blob/master/drivers/middleware/stdlib/middleware.go#L36
func bunrouterLimitMiddleware(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		if Config.Verbose > 0 {
			log.Println("limiter middleware check")
		}
		r := req.Request
		key := limiterMiddleware.KeyGetter(r)
		if limiterMiddleware.ExcludedKey != nil && limiterMiddleware.ExcludedKey(key) {
			return next(w, req)
		}

		context, err := limiterMiddleware.Limiter.Get(r.Context(), key)
		if err != nil {
			limiterMiddleware.OnError(w, r, err)
			return err
		}

		w.Header().Add("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		w.Header().Add("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		if context.Reached {
			limiterMiddleware.OnLimitReached(w, r)
			return nil
		}
		// execute next ServeHTTP middleware/step
		return next(w, req)
	}
}
