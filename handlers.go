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
	Message        string `json:"message"`          // response message
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
	tdir := fmt.Sprintf("%s/templates", Config.StaticDir)
	page := templates.Tmpl(tdir, tmpl, tmplData)
	return page
}

// helper function to generate JSON response
func httpResponse(w http.ResponseWriter, r *http.Request, tmpl TmplRecord) {
	httpCode := tmpl.Int("HttpCode")
	code := tmpl.Int("Code")
	top := tmpl.String("Top")
	bottom := tmpl.String("Bottom")
	page := tmpl.String("Page")
	msg := tmpl.String("Content")
	if r.Header.Get("Accept") != "application/json" {
		w.WriteHeader(httpCode)
		w.Write([]byte(top + page + bottom))
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
		Message:        msg,
		Error:          tmpl.String("Error"),
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
	tmpl := make(TmplRecord)
	tmpl["Base"] = Config.Base
	tmpl["ServerInfo"] = info()
	data, err := json.MarshalIndent(hrec, "", "   ")
	if err != nil {
		data = []byte(err.Error())
	}
	if Config.Verbose > 0 {
		log.Println("ERROR:", data)
	}
	if r.Header.Get("Accept") == "application/json" {
		w.WriteHeader(httpCode)
		w.Write([]byte(data))
		return
	}
	tmpl["Content"] = fmt.Sprintf("<pre>%s</pre>", data)
	page := tmplPage("error.tmpl", tmpl)
	w.WriteHeader(httpCode)
	w.Write([]byte(page))
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
	model := r.FormValue("model")
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
	tmpl["Top"] = top
	tmpl["Bottom"] = bottom

	// handle web GET request to upload page
	if r.Method == "GET" {
		page := tmplPage("upload.tmpl", tmpl)
		w.Write([]byte(top + page + bottom))
		return
	}

	// check if we provided with proper form data
	if !formData(r) {
		httpError(w, r, BadRequest, errors.New("unable to get form data"), http.StatusBadRequest)
		return
	}

	// handle upload POST requests
	var rec Record
	var err error
	if strings.Contains(r.URL.Path, "/model") {
		// POST request to /model/:model/upload API
		rec, err = modelRecord(r)
		if err != nil {
			httpError(w, r, BadRequest, err, http.StatusBadRequest)
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
				httpError(w, r, BadRequest, err, http.StatusBadRequest)
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

	// default response attribute
	var page string
	content := fmt.Sprintf("ML model %s has been successfully uploaded to MLHub", rec.Model)
	tfile := "success.tmpl"

	// perform upload action
	err = Upload(rec, r)
	if Config.Verbose > 0 {
		log.Printf("Upload %+v, error: %v", rec, err)
	}

	// prepare proper response
	if err != nil {
		content = fmt.Sprintf("Unable to insert ML model %s, error: %v", rec.Model, err)
		tfile = "error.tmpl"
		tmpl["Content"] = template.HTML(content)
		tmpl["Error"] = err.Error()
		tmpl["HttpCode"] = http.StatusBadRequest
		tmpl["Code"] = InsertError
		httpResponse(w, r, tmpl)
		return
	}
	tmpl["Content"] = template.HTML(content)
	page = tmplPage(tfile, tmpl)
	tmpl["Page"] = page
	httpResponse(w, r, tmpl)
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
