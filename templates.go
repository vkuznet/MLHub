package main

// templates module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"strconv"
)

// TmplRecord represent template record
type TmplRecord map[string]interface{}

// String converts given value for provided key to string data-type
func (t TmplRecord) String(key string) string {
	if v, ok := t[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// Int converts given value for provided key to int data-type
func (t TmplRecord) Int(key string) int {
	if v, ok := t[key]; ok {
		if val, err := strconv.Atoi(fmt.Sprintf("%v", v)); err == nil {
			return val
		} else {
			log.Println("ERROR:", err)
		}
	}
	return -1
}

// consume list of templates and release their full path counterparts
func fileNames(tdir string, filenames ...string) []string {
	flist := []string{}
	for _, fname := range filenames {
		flist = append(flist, filepath.Join(tdir, fname))
	}
	return flist
}

// parse template with given data
func parseTmpl(tdir, tmpl string, data interface{}) string {
	buf := new(bytes.Buffer)
	filenames := fileNames(tdir, tmpl)
	funcMap := template.FuncMap{
		// The name "oddFunc" is what the function will be called in the template text.
		"oddFunc": func(i int) bool {
			if i%2 == 0 {
				return true
			}
			return false
		},
		// The name "inListFunc" is what the function will be called in the template text.
		"inListFunc": func(a string, list []string) bool {
			check := 0
			for _, b := range list {
				if b == a {
					check += 1
				}
			}
			if check != 0 {
				return true
			}
			return false
		},
	}
	t := template.Must(template.New(tmpl).Funcs(funcMap).ParseFiles(filenames...))
	err := t.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// Templates structure
type Templates struct {
	html string
}

// Tmpl method for ServerTemplates structure
func (q Templates) Tmpl(tdir, tfile string, tmplData map[string]interface{}) string {
	if q.html != "" {
		return q.html
	}
	q.html = parseTmpl(tdir, tfile, tmplData)
	return q.html
}
