package main

// handlers module holds all HTTP handlers functions
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/uptrace/bunrouter"
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

// helper function to get model name from http request
func getModel(r *http.Request) (string, bool) {
	vars := mux.Vars(r)
	model, ok := vars["model"]
	if !ok { // no gorilla/mux, try bunrouter params map
		params := bunrouter.ParamsFromContext(r.Context())
		model, ok = params.Map()["model"]
	}
	return model, ok
}

// helper function to parse given template and return HTML page
func tmplPage(tmpl string, tmplData TmplRecord) string {
	if tmplData == nil {
		tmplData = make(TmplRecord)
	}
	var templates Templates
	tdir := fmt.Sprintf("%s/templates", Config.StaticDir)
	page := templates.Tmpl(tdir, tmpl, tmplData)
	return page
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

// helper function to check record attributes
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
	rec, err := modelRecord(r)
	if err != nil {
		httpError(w, r, BadRequest, err, http.StatusBadRequest)
	}
	if backend, ok := Config.MLBackends[rec.Type]; ok {
		path := r.RequestURI
		bPath := strings.Replace(path, fmt.Sprintf("/model/%s", rec.Model), "", -1)
		uri := fmt.Sprintf("%s", backend.URI)
		rurl := uri + bPath
		if Config.Verbose > 0 {
			log.Printf("get predictions from %s model at %s", rec.Model, rurl)
		}
		data, err := Predict(rurl, rec.Model, r)
		if err == nil {
			w.Write(data)
		} else {
			httpError(w, r, BadRequest, err, http.StatusBadRequest)
		}
	} else {
		msg := fmt.Sprintf("no ML backed record found for %s", rec.Type)
		httpError(w, r, BadRequest, errors.New(msg), http.StatusBadRequest)
	}
}

// DownloadHandler handles download action of ML model from back-end server
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && !strings.Contains(r.URL.Path, "/model") {
		fname := fmt.Sprintf("%s/md/download.md", Config.StaticDir)
		content, err := mdToHTML(fname)
		if err != nil {
			httpError(w, r, FileIOError, err, http.StatusInternalServerError)
			return
		}

		tmpl := make(TmplRecord)
		tmpl["Title"] = "MLHub download"
		tmpl["Content"] = template.HTML(content)
		tmpl["Base"] = Config.Base
		tmpl["ServerInfo"] = info()

		page := tmplPage("download.tmpl", tmpl)
		top := tmplPage("top.tmpl", tmpl)
		bottom := tmplPage("bottom.tmpl", tmpl)
		w.Write([]byte(top + page + bottom))
		return
	}

	// CLI /model/:mname/download
	model := r.FormValue("name")
	mlType := r.FormValue("type")
	version := r.FormValue("version")
	// check if record exist in MetaData database
	spec := bson.M{"model": model, "type": mlType, "version": version}
	records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	if err != nil {
		httpError(w, r, BadRequest, err, http.StatusBadRequest)
	}
	if len(records) != 1 {
		msg := fmt.Sprintf("Too many records for provide model=%s type=%s version=%s", model, mlType, version)
		httpError(w, r, BadRequest, errors.New(msg), http.StatusBadRequest)
	}
	rec := records[0]
	// form link to download the model bundle
	downloadURL := fmt.Sprintf("/bundles/%s/%s/%s/%s", mlType, model, version, rec.Bundle)
	if Config.Verbose > 0 {
		log.Println("download", downloadURL)
	}
	http.Redirect(w, r, downloadURL, http.StatusSeeOther)
}

// UploadHandler handles upload action of ML model to back-end server
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := make(TmplRecord)
	tmpl["Title"] = "MLHub upload"
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	top := tmplPage("top.tmpl", tmpl)
	bottom := tmplPage("bottom.tmpl", tmpl)
	if r.Method == "GET" {
		page := tmplPage("upload.tmpl", tmpl)
		w.Write([]byte(top + page + bottom))
		return
	}

	// This handler processes two types of POST HTTP requests
	// POST client for /model/:model/upload API
	if strings.Contains(r.URL.Path, "/model") {
		rec, err := modelRecord(r)
		if err != nil {
			httpError(w, r, BadRequest, err, http.StatusBadRequest)
		}
		// check if we provided with proper form data
		if !formData(r) {
			httpError(w, r, BadRequest, errors.New("unable to get form data"), http.StatusBadRequest)
			return
		}
		err = Upload(rec, r)
		if err != nil {
			httpError(w, r, BadRequest, err, http.StatusInternalServerError)
		}
	}

	// - web HTML form for /upload API
	model := r.FormValue("name")
	mlType := r.FormValue("type")
	version := r.FormValue("version")
	reference := r.FormValue("reference")
	discipline := r.FormValue("discipline")
	description := r.FormValue("description")
	if Config.Verbose > 0 {
		log.Printf("UploadHandler form: model=%s type=%s version=%s reference=%s discipline=%s description=%s", model, mlType, version, reference, discipline, description)
	}
	// parse incoming HTTP request multipart form
	err := r.ParseMultipartForm(32 << 20) // maxMemory
	if err == nil {
		if file, handler, err := r.FormFile("file"); err == nil {
			defer file.Close()
			modelDir := fmt.Sprintf("%s/%s/%s/%s", Config.StorageDir, mlType, model, version)
			err := os.MkdirAll(modelDir, 0755)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fname := filepath.Join(modelDir, handler.Filename)
			dst, err := os.Create(fname)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer dst.Close()
			if _, err := io.Copy(dst, file); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		tmpl["Content"] = fmt.Sprintf("Unable to insert ML model <b>%s</b>, error: %v", model, err)
		page := tmplPage("error.tmpl", tmpl)
		w.Write([]byte(top + page + bottom))
		return
	}

	// we got HTML form request
	content := fmt.Sprintf("ML model <b>%s</b> has been successfully uploaded to MLHub", model)
	tmpl["Content"] = template.HTML(content)
	rec := Record{
		Model:       model,
		Type:        mlType,
		Version:     version,
		Description: description,
		Discipline:  discipline,
		Reference:   reference,
	}
	// insert record into MetaData database
	records := []Record{rec}
	err = MongoUpsert(Config.DBName, Config.DBColl, records)
	var page string
	if err == nil {
		page = tmplPage("success.tmpl", tmpl)
	} else {
		tmpl["Content"] = fmt.Sprintf("Unable to insert ML model <b>%s</b>, error: %v", model, err)
		page = tmplPage("error.tmpl", tmpl)
	}
	w.Write([]byte(top + page + bottom))
	return
}

// GetHandler handles GET HTTP requests
func GetHandler(w http.ResponseWriter, r *http.Request) {
	model, ok := getModel(r)
	if ok {
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
	// if we are here we'll show HTTP content
	tmpl := make(TmplRecord)
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	page := tmplPage("index.tmpl", tmpl)
	w.Write([]byte(page))
}

// helper function either to create/upsert or update record
func addRecord(r *http.Request, update bool) error {
	// TODO: add code to create ML model on backend
	// so far the code below only creates ML model info in MetaData database
	model, ok := getModel(r)
	if ok {
		if Config.Verbose > 0 {
			log.Printf("update ML model %s", model)
		}
		// parse input JSON body
		decoder := json.NewDecoder(r.Body)
		var rec Record
		err := decoder.Decode(&rec)
		if err != nil {
			return err
		}
		if err := checkRecord(rec, model); err != nil {
			return err
		}
		if update {
			// update ML meta-data
			spec := bson.M{"model": model}
			meta := bson.M{"model": model, "type": rec.Type, "meta_data": rec.MetaData}
			err = MongoUpdate(Config.DBName, Config.DBColl, spec, meta)
		} else {
			// insert ML meta-data
			records := []Record{rec}
			err = MongoUpsert(Config.DBName, Config.DBColl, records)
		}
		return err
	}
	msg := fmt.Sprintf("unable to get model HTTP parameter")
	return errors.New(msg)
}

// PostHandler handles POST HTTP requests,
// this request will create and upload ML models to backend server(s)
func PostHandler(w http.ResponseWriter, r *http.Request) {
	err := addRecord(r, false)
	if err != nil {
		httpError(w, r, BadRequest, err, http.StatusBadRequest)
	}
}

// PutHandler handles PUT HTTP requests, this request will
// update ML model in backend or MetaData database
func PutHandler(w http.ResponseWriter, r *http.Request) {
	err := addRecord(r, true)
	if err != nil {
		httpError(w, r, BadRequest, err, http.StatusBadRequest)
	}
}

// GetHandler handles GET HTTP requests, this request will
// delete ML model in backend and MetaData database
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	model, ok := getModel(r)
	if ok {
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
	// TODO: Add parameters for /models endpoint, eg q=query, limit, idx for pagination
	spec := bson.M{}
	records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	if err != nil {
		msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
		httpError(w, r, DatabaseError, errors.New(msg), http.StatusInternalServerError)
		return
	}
	log.Println("### request", r.Header.Get("Accept"), r)
	if r.Header.Get("Accept") == "application/json" {
		data, err := json.Marshal(records)
		if err != nil {
			msg := fmt.Sprintf("unable to marshal data, error=%v", err)
			httpError(w, r, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	tmpl := make(TmplRecord)
	tmpl["Title"] = "MLHub models"
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	tmpl["Records"] = records
	page := tmplPage("models.tmpl", tmpl)
	top := tmplPage("top.tmpl", tmpl)
	bottom := tmplPage("bottom.tmpl", tmpl)
	w.Write([]byte(top + page + bottom))
}

// InferenceHandler handles status of MLHub server
func InferenceHandler(w http.ResponseWriter, r *http.Request) {
	fname := fmt.Sprintf("%s/md/inference.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, FileIOError, err, http.StatusInternalServerError)
		return
	}

	tmpl := make(TmplRecord)
	tmpl["Title"] = "MLHub inference"
	tmpl["Content"] = template.HTML(content)
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()

	page := tmplPage("inference.tmpl", tmpl)
	top := tmplPage("top.tmpl", tmpl)
	bottom := tmplPage("bottom.tmpl", tmpl)
	w.Write([]byte(top + page + bottom))
}

// DocsHandler handles status of MLHub server
func DocsHandler(w http.ResponseWriter, r *http.Request) {
	fname := fmt.Sprintf("%s/md/docs.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, FileIOError, err, http.StatusInternalServerError)
		return
	}
	tmpl := make(TmplRecord)
	tmpl["Title"] = "MLHub documentation"
	tmpl["Content"] = template.HTML(content)
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	page := tmplPage("docs.tmpl", tmpl)
	top := tmplPage("top.tmpl", tmpl)
	bottom := tmplPage("bottom.tmpl", tmpl)
	w.Write([]byte(top + page + bottom))
}

// DomainsHandler handles status of MLHub server
func DomainsHandler(w http.ResponseWriter, r *http.Request) {
	fname := fmt.Sprintf("%s/md/domains.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, FileIOError, err, http.StatusInternalServerError)
		return
	}
	tmpl := make(TmplRecord)
	tmpl["Title"] = "MLHub scientific domains (disciplines)"
	tmpl["Content"] = template.HTML(content)
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	page := tmplPage("domains.tmpl", tmpl)
	top := tmplPage("top.tmpl", tmpl)
	bottom := tmplPage("bottom.tmpl", tmpl)
	w.Write([]byte(top + page + bottom))
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
