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
	"log"
	"net/http"
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

// HTTPResponse rpresents HTTP JSON response
type HTTPResponse struct {
	Method         string `json:"method"`           // HTTP method
	Path           string `json:"path"`             // URL path
	UserAgent      string `json:"user_agent"`       // http user-agent field
	XForwardedHost string `json:"x_forwarded_host"` // http.Request X-Forwarded-Host
	XForwardedFor  string `json:"x_forwarded_for"`  // http.Request X-Forwarded-For
	RemoteAddr     string `json:"remote_addr"`      // http.Request remote address
	HTTPCode       int    `json:"http_code"`        // HTTP error code
	Code           int    `json:"code"`             // server status code
	Reason         string `json:"reason"`           // error code reason
	Timestamp      string `json:"timestamp"`        // timestamp of the error
	Response       string `json:"response"`         // response message
	Error          string `json:"error"`            // error message
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
	page := templates.Tmpl(tmpl, tmplData)
	//     tdir := fmt.Sprintf("%s/templates", Config.StaticDir)
	//     page := templates.TmplFile(tdir, tmpl, tmplData)
	return page
}

// helper function to generate JSON response
func httpResponse(w http.ResponseWriter, r *http.Request, tmpl TmplRecord) {
	httpCode := tmpl.Int("HttpCode")
	code := tmpl.Int("Code")
	content := tmpl.String("Content")
	if r.Header.Get("Accept") != "application/json" {
		top := tmpl.String("Top")
		bottom := tmpl.String("Bottom")
		tfile := tmpl.String("Template")
		page := tmplPage(tfile, tmpl)
		if httpCode != 0 {
			w.WriteHeader(httpCode)
		}
		if tfile == "index.tmpl" {
			w.Write([]byte(page))
		} else {
			w.Write([]byte(top + page + bottom))
		}
		return
	}
	if httpCode == 0 {
		httpCode = http.StatusOK
	}
	hrec := HTTPResponse{
		Method:         r.Method,
		Path:           r.RequestURI,
		RemoteAddr:     r.RemoteAddr,
		XForwardedFor:  r.Header.Get("X-Forwarded-For"),
		XForwardedHost: r.Header.Get("X-Forwarded-Host"),
		UserAgent:      r.Header.Get("User-agent"),
		Timestamp:      time.Now().String(),
		Code:           code,
		Reason:         errorMessage(code),
		HTTPCode:       httpCode,
		Response:       content,
		Error:          tmpl.Error(),
	}
	if Config.Verbose > 0 {
		log.Printf("HTTPResponse: %+v", hrec)
	}
	data, err := json.MarshalIndent(hrec, "", "   ")
	if err != nil {
		data = []byte(err.Error())
	}
	w.WriteHeader(httpCode)
	w.Write([]byte(data))
}

// helper function to provide standard HTTP error reply
func httpError(w http.ResponseWriter, r *http.Request, tmpl TmplRecord, code int, err error, httpCode int) {
	tmpl["Code"] = code
	tmpl["Error"] = err
	tmpl["HttpCode"] = httpCode
	tmpl["Content"] = err.Error()
	tmpl["Template"] = "error.tmpl"
	httpResponse(w, r, tmpl)
}

// helper function to make initial template struct
func makeTmpl(title string) TmplRecord {
	tmpl := make(TmplRecord)
	tmpl["Title"] = title
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	tmpl["Top"] = tmplPage("top.tmpl", tmpl)
	tmpl["Bottom"] = tmplPage("bottom.tmpl", tmpl)
	return tmpl
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
	tmpl := makeTmpl("MLHub predict")
	rec, err := modelRecord(r)
	if err != nil {
		httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
		return
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
			httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
			return
		}
	} else {
		msg := fmt.Sprintf("no ML backed record found for %s", rec.Type)
		httpError(w, r, tmpl, BadRequest, errors.New(msg), http.StatusBadRequest)
		return
	}
}

// DownloadHandler handles download action of ML model from back-end server
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub download")
	if r.Method == "GET" && !strings.Contains(r.URL.Path, "/model") {
		fname := fmt.Sprintf("%s/md/download.md", Config.StaticDir)
		content, err := mdToHTML(fname)
		if err != nil {
			httpError(w, r, tmpl, FileIOError, err, http.StatusInternalServerError)
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
	model := r.FormValue("model")
	mlType := r.FormValue("type")
	version := r.FormValue("version")
	// check if record exist in MetaData database
	spec := bson.M{"model": model, "type": mlType, "version": version}
	records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	if err != nil {
		httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
		return
	}
	if len(records) != 1 {
		msg := fmt.Sprintf("Too many records for provide model=%s type=%s version=%s", model, mlType, version)
		httpError(w, r, tmpl, BadRequest, errors.New(msg), http.StatusBadRequest)
		return
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
	tmpl := makeTmpl("MLHub upload")

	// handle web GET request to upload page
	if r.Method == "GET" {
		tmpl["Template"] = "upload.tmpl"
		httpResponse(w, r, tmpl)
		return
	}

	// check if we provided with proper form data
	if !formData(r) {
		httpError(w, r, tmpl, BadRequest, errors.New("unable to get form data"), http.StatusBadRequest)
		return
	}

	// handle upload POST requests
	var rec Record
	var err error
	if strings.Contains(r.URL.Path, "/model") {
		// POST request to /model/:model/upload API
		rec, err = modelRecord(r)
		if err != nil {
			httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
			return
		}
	} else {
		// POST web form request to /upload API
		model := r.FormValue("model")
		mlType := r.FormValue("type")
		bundle := r.FormValue("file")
		version := r.FormValue("version")
		reference := r.FormValue("reference")
		discipline := r.FormValue("discipline")
		description := r.FormValue("description")

		// get file name bundle
		if bundle == "" {
			// parse incoming HTTP request multipart form
			err := r.ParseMultipartForm(32 << 20) // maxMemory
			if err != nil {
				httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
				return
			}
			for _, vals := range r.MultipartForm.File {
				for _, fh := range vals {
					bundle = fh.Filename
				}
			}
		}

		// we got HTML form request
		rec = Record{
			Model:       model,
			Type:        mlType,
			Version:     version,
			Description: description,
			Discipline:  discipline,
			Reference:   reference,
			Bundle:      bundle,
		}
	}

	// perform upload action
	err = Upload(rec, r)
	if err != nil {
		httpError(w, r, tmpl, InsertError, err, http.StatusBadRequest)
		return
	}
	content := fmt.Sprintf("ML model %s has been successfully uploaded to MLHub", rec.Model)
	tmpl["Content"] = template.HTML(content)
	tmpl["Template"] = "success.tmpl"
	httpResponse(w, r, tmpl)
}

// GetHandler handles GET HTTP requests
func GetHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub")
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
			httpError(w, r, tmpl, DatabaseError, errors.New(msg), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(records)
		if err != nil {
			msg := fmt.Sprintf("unable to marshal data, error=%v", err)
			httpError(w, r, tmpl, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	// if we are here we'll show HTTP content
	tmpl["Template"] = "index.tmpl"
	httpResponse(w, r, tmpl)
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
	tmpl := makeTmpl("MLHub POST API")
	err := addRecord(r, false)
	if err != nil {
		httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
		return
	}
	tmpl["Template"] = "success.tmpl"
	httpResponse(w, r, tmpl)
}

// PutHandler handles PUT HTTP requests, this request will
// update ML model in backend or MetaData database
func PutHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub PUT API")
	err := addRecord(r, true)
	if err != nil {
		httpError(w, r, tmpl, BadRequest, err, http.StatusBadRequest)
		return
	}
	tmpl["Template"] = "success.tmpl"
	httpResponse(w, r, tmpl)
}

// GetHandler handles GET HTTP requests, this request will
// delete ML model in backend and MetaData database
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub DELETE API")
	model, ok := getModel(r)
	if ok {
		if Config.Verbose > 0 {
			log.Printf("delete ML model %s", model)
		}
		// delete ML model in MetaData database
		spec := bson.M{"name": model}
		err := MongoRemove(Config.DBName, Config.DBColl, spec)
		if err != nil {
			httpError(w, r, tmpl, DatabaseError, err, http.StatusInternalServerError)
			return
		}
		tmpl["Template"] = "success.tmpl"
		httpResponse(w, r, tmpl)
		return
	}
	httpError(w, r, tmpl, BadRequest, errors.New("no model name is provided"), http.StatusBadRequest)
}

// ModelsHandler provides information about registered ML models
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub models")
	// TODO: Add parameters for /models endpoint, eg q=query, limit, idx for pagination
	spec := bson.M{}
	records, err := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
	if err != nil {
		msg := fmt.Sprintf("unable to get meta-data, error=%v", err)
		httpError(w, r, tmpl, DatabaseError, errors.New(msg), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("Accept") == "application/json" {
		data, err := json.Marshal(records)
		if err != nil {
			msg := fmt.Sprintf("unable to marshal data, error=%v", err)
			httpError(w, r, tmpl, JsonMarshal, errors.New(msg), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	tmpl["Records"] = records
	tmpl["Template"] = "models.tmpl"
	httpResponse(w, r, tmpl)
}

// InferenceHandler handles status of MLHub server
func InferenceHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub inference")
	fname := fmt.Sprintf("%s/md/inference.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, tmpl, FileIOError, err, http.StatusInternalServerError)
		return
	}

	tmpl["Content"] = template.HTML(content)
	tmpl["Template"] = "inference.tmpl"
	httpResponse(w, r, tmpl)
}

// DocsHandler handles status of MLHub server
func DocsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub documentation")
	fname := fmt.Sprintf("%s/md/docs.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, tmpl, FileIOError, err, http.StatusInternalServerError)
		return
	}
	tmpl["Content"] = template.HTML(content)
	tmpl["Template"] = "docs.tmpl"
	httpResponse(w, r, tmpl)
}

// DomainsHandler handles status of MLHub server
func DomainsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := makeTmpl("MLHub scientific domains (disciplines)")
	fname := fmt.Sprintf("%s/md/domains.md", Config.StaticDir)
	content, err := mdToHTML(fname)
	if err != nil {
		httpError(w, r, tmpl, FileIOError, err, http.StatusInternalServerError)
		return
	}
	tmpl["Content"] = template.HTML(content)
	tmpl["Template"] = "domains.tmpl"
	httpResponse(w, r, tmpl)
}

// StatusHandler handles status of MLHub server
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement status API from all backend servers
	tmpl := makeTmpl("MLHub predict")
	tmpl["Template"] = "status.tmpl"
	httpResponse(w, r, tmpl)
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
		tmpl := makeTmpl("MLHub request")
		msg := fmt.Sprintf("Unsupport HTTP method %s", r.Method)
		httpError(w, r, tmpl, BadRequest, errors.New(msg), http.StatusInternalServerError)
	}
}
