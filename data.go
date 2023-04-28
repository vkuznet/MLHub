package main

// data module holds all data representations used in our package
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// Configuration stores server configuration parameters
type Configuration struct {
	Port                int        `json:"port"`                   // server port number
	RootCAs             string     `json:"rootCAs"`                // server Root CAs path
	Base                string     `json:"base"`                   // base URL
	LogFile             string     `json:"log_file"`               // server log file
	XForwardedHost      string     `json:"X-Forwarded-Host"`       // X-Forwarded-Host field of HTTP request
	XContentTypeOptions string     `json:"X-Content-Type-Options"` // X-Content-Type-Options option
	Verbose             int        `json:"verbose"`                // verbose output
	ServerCrt           string     `json:"server_cert"`            // server certificate
	ServerKey           string     `json:"server_key"`             // server certificate
	DomainNames         []string   `json:"domain_names"`           // list of domain names to use for LetsEncrypt
	StaticDir           string     `json:"staticDir"`              // speficy static dir location
	LimiterPeriod       string     `json:"rate"`                   // github.com/ulule/limiter rate value
	DBURI               string     `json:"db_uri"`                 // meta-data server URI
	DBName              string     `json:"db_name"`                // meta-data database name
	DBColl              string     `json:"db_coll"`                // meta-data database collection name
	MLBackends          MLBackends `json:"backends"`               // ML backends
}

// MLTypes defines supported ML data types
var MLTypes = []string{"TensorFlow", "PyTorch", "ScikitLearn"}

// MLBackend represents ML backend engine
type MLBackend struct {
	Name string   // ML backend name, e.g. TFaaS
	Type string   // ML backebd type, e.g. TensorFlow
	URI  string   // ML backend URI, e.g. http://localhost:port
	APIs []string // ML backend APIs for upload and delete
}

// Upload performs upload of the given data to upstream ML backend
func (m *MLBackend) Upload(data []byte) error {
	return nil
}

// Download downloads ML model from backend server
func (m *MLBackend) Download(model string) ([]byte, error) {
	return []byte{}, nil
}

// Delete performs delete action of the ML model on ML backend
func (m *MLBackend) Delete(model string) error {
	return nil
}

// MLBackends represents map of ML backends records
type MLBackends map[string]MLBackend
