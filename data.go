package main

// data module holds all data representations used in our package
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// Configuration stores server configuration parameters
type Configuration struct {
	Port                int      `json:"port"`                   // server port number
	RootCAs             string   `json:"rootCAs"`                // server Root CAs path
	Base                string   `json:"base"`                   // base URL
	LogFile             string   `json:"log_file"`               // server log file
	XForwardedHost      string   `json:"X-Forwarded-Host"`       // X-Forwarded-Host field of HTTP request
	XContentTypeOptions string   `json:"X-Content-Type-Options"` // X-Content-Type-Options option
	Verbose             int      `json:"verbose"`                // verbose output
	ServerCrt           string   `json:"server_cert"`            // server certificate
	ServerKey           string   `json:"server_key"`             // server certificate
	DomainNames         []string `json:"domain_names"`           // list of domain names to use for LetsEncrypt
	StaticDir           string   `json:"staticDir"`              // speficy static dir location
	LimiterPeriod       string   `json:"rate"`                   // github.com/ulule/limiter rate value
	URI                 string   `json:"uri"`                    // meta-data server URI
}
