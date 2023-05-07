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
	return 0
}

// Error returns error string
func (t TmplRecord) Error() string {
	if v, ok := t["Error"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// Templates structure
type Templates struct {
	html string
}

// Tmpl method for ServerTemplates structure
func (q Templates) Tmpl(tfile string, tmplData map[string]interface{}) string {
	if q.html != "" {
		return q.html
	}

	// get template from embed.FS
	filenames := []string{"static/templates/" + tfile}
	t := template.Must(template.New(tfile).ParseFS(StaticFs, filenames...))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, tmplData)
	if err != nil {
		panic(err)
	}
	q.html = buf.String()
	return q.html
}
