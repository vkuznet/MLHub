package main

// client functions for ML backends
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gomarkdown/markdown"
	mhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/uptrace/bunrouter"
)

// Predict function fetches prediction for given uri, model and client's
// HTTP request. Code is based on the following example:
// https://golangbyexample.com/http-mutipart-form-body-golang/
func Predict(uri string, rec Record, r *http.Request) ([]byte, error) {
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
	writer.WriteField("model", rec.Model)

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

	// for TFaaS we need additional end-point path if we query image prediction
	if r.FormValue("name") != "image" && rec.Type == "TensorFlow" {
		uri += "/image"
	}
	if Config.Verbose > 0 {
		log.Printf("Predict uri=%s HTTP request %+v", uri, r)
	}

	// form HTTP request
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

// Upload function uploads record to MetaData database, then
// uploads file to server storage, and finally to ML backend
func Upload(rec Record, r *http.Request) error {
	err := uploadRecord(rec)
	if err != nil {
		return err
	}
	err = uploadStorage(rec, r)
	if err != nil {
		return err
	}
	err = uploadBundle(rec, r)
	if err != nil {
		return err
	}
	return nil
}

// helper function to upload bundle tarball to ML backend
func uploadRecord(rec Record) error {
	// insert record into MetaData database
	if Config.Verbose > 0 {
		log.Printf("uploadRecord %+v", rec)
	}
	err := metadata.Insert(rec)
	return err
}

// helper function to upload bundle to server storage
func uploadStorage(rec Record, r *http.Request) error {
	if Config.Verbose > 0 {
		log.Printf("uploadStorage %+v", rec)
	}
	// parse incoming HTTP request multipart form
	err := r.ParseMultipartForm(32 << 20) // maxMemory
	if err != nil {
		return err
	}
	// extract file from HTTP request form
	file, handler, err := r.FormFile("file")
	if err != nil {
		return err
	}

	defer file.Close()
	modelDir := fmt.Sprintf("%s/%s/%s/%s", Config.StorageDir, rec.Type, rec.Model, rec.Version)
	err = os.MkdirAll(modelDir, 0755)
	if err != nil {
		return err
	}
	fname := filepath.Join(modelDir, handler.Filename)
	dst, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return err
	}
	return nil
}

// helper function to upload bundle tarball to ML backend
func uploadBundle(rec Record, r *http.Request) error {
	if rec.Type == "TensorFlow" {
		return uploadBundleTFaaS(rec, r)
	} else if rec.Type == "PyTorch" {
		return uploadBundleTorch(rec, r)
	} else if rec.Type == "ScikitLearn" {
		return uploadBundleScikit(rec, r)
	}
	msg := fmt.Sprintf("upload for %s backend is not implemented", rec.Type)
	return errors.New(msg)
}

// helper functiont to upload bundle to TFaaS backend
func uploadBundleTFaaS(rec Record, r *http.Request) error {
	// curl -v -X POST -H"Content-Encoding: gzip" -H"content-type: application/octet-stream" --data-binary @$model_tarball $turl/upload
	backend, ok := Config.MLBackends[rec.Type]
	if !ok {
		msg := fmt.Sprintf("upload for %s backend is not implemented", rec.Type)
		return errors.New(msg)
	}

	// form backe URI
	uri := fmt.Sprintf("%s/upload", backend.URI)
	if Config.Verbose > 0 {
		log.Printf("upload model %s bundle to %s", rec.Model, uri)
	}
	// parse incoming HTTP request multipart form
	err := r.ParseMultipartForm(32 << 20) // maxMemory

	// construct proper request body
	var body io.Reader
	for _, vals := range r.MultipartForm.File {
		for _, fh := range vals {
			file, err := fh.Open()
			if err != nil {
				return err
			}
			body = io.NopCloser(file)
		}
	}

	// make HTTP request to remote TFaaS server
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/octet-stream")
	if Config.Verbose > 0 {
		log.Printf("New request %+v", req)
	}
	rsp, err := client.Do(req)
	if Config.Verbose > 0 {
		log.Println("TFaaS response", rsp)
	}
	if err == nil {
		// check response status code
		if rsp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("TFaaS response status %s", rsp.Status)
			err = errors.New(msg)
		}
	}
	return err
}

// helper functiont to upload bundle to Torch backend
func uploadBundleTorch(rec Record, r *http.Request) error {
	return errors.New("upload for TorchServer backend is not implemented")
}

// helper functiont to upload bundle to Scikit backend
func uploadBundleScikit(rec Record, r *http.Request) error {
	return errors.New("upload for ScikitLearn backend is not implemented")
}

// helper function to get ML record for given HTTP request
func modelRecord(r *http.Request) (Record, error) {
	var rec Record

	// look-up model from HTTP request parameters
	params := bunrouter.ParamsFromContext(r.Context())
	model, _ := params.Map()["model"]

	// final try from the web form (HTTP POST request)
	if model == "" {
		model = r.FormValue("model")
	}
	if model == "" {
		msg := fmt.Sprintf("Unable to find model in MetaData database")
		log.Printf("ERROR: %s, HTTP request %+v", msg, r)
		return rec, errors.New(msg)
	}
	version, _ := params.Map()["version"]
	if version == "" {
		version = r.FormValue("version")
	}
	mtype, _ := params.Map()["type"]
	if mtype == "" {
		mtype = r.FormValue("mtype")
	}

	if Config.Verbose > 0 {
		log.Printf("get meta-data for model=%s type=%s version=%s", model, mtype, version)
	}
	// get ML meta-data
	records, err := metadata.Records(model, mtype, version)
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

// helper function to parse given markdown file and return HTML content
func mdToHTML(fname string) (string, error) {
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var md []byte
	md, err = io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := mhtml.CommonFlags | mhtml.HrefTargetBlank
	opts := mhtml.RendererOptions{Flags: htmlFlags}
	renderer := mhtml.NewRenderer(opts)
	content := markdown.Render(doc, renderer)
	//     return html.EscapeString(string(content)), nil
	return string(content), nil
}
