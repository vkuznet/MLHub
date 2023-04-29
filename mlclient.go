package main

// client functions for ML backends
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"
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
