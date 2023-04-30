package main

// data module holds all data representations used in our package
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// Configuration stores server configuration parameters
type Configuration struct {
	// server parts
	Base    string `json:"base"`     // base URL
	LogFile string `json:"log_file"` // server log file
	Port    int    `json:"port"`     // server port number
	Verbose int    `json:"verbose"`  // verbose output

	// proxy parts
	XForwardedHost      string `json:"X-Forwarded-Host"`       // X-Forwarded-Host field of HTTP request
	XContentTypeOptions string `json:"X-Content-Type-Options"` // X-Content-Type-Options option

	// server parts
	RootCAs       string   `json:"rootCAs"`      // server Root CAs path
	ServerCrt     string   `json:"server_cert"`  // server certificate
	ServerKey     string   `json:"server_key"`   // server certificate
	DomainNames   []string `json:"domain_names"` // LetsEncrypt domain names
	StaticDir     string   `json:"static_dir"`   // speficy static dir location
	LimiterPeriod string   `json:"rate"`         // limiter rate value

	// MetaData parts
	DBURI      string     `json:"db_uri"`   // meta-data server URI
	DBName     string     `json:"db_name"`  // meta-data database name
	DBColl     string     `json:"db_coll"`  // meta-data database collection
	MLBackends MLBackends `json:"backends"` // ML backends
}

// MLTypes defines supported ML data types
var MLTypes = []string{"TensorFlow", "PyTorch", "ScikitLearn"}

// MLBackend represents ML backend engine
type MLBackend struct {
	Name string `json:"name"` // ML backend name, e.g. TFaaS
	Type string `json:"type"` // ML backebd type, e.g. TensorFlow
	URI  string `json:"uri"`  // ML backend URI, e.g. http://localhost:port
}

// Predict performs predict action on upstream ML backend
func (m *MLBackend) Predict(data []byte) ([]byte, error) {
	return []byte{}, nil
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
