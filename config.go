package main

// config module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Configuration stores server configuration parameters
type Configuration struct {
	// web server parts
	Base      string `json:"base"`       // base URL
	LogFile   string `json:"log_file"`   // server log file
	Port      int    `json:"port"`       // server port number
	Verbose   int    `json:"verbose"`    // verbose output
	StaticDir string `json:"static_dir"` // speficy static dir location

	// OAuth parts
	OAuthHost    string `json:"oauth_host"`    // OAuthHost name
	ClientID     string `json:"client_id"`     // client id
	ClientSecret string `json:"client_secret"` // client secret

	// proxy parts
	XForwardedHost      string `json:"X-Forwarded-Host"`       // X-Forwarded-Host field of HTTP request
	XContentTypeOptions string `json:"X-Content-Type-Options"` // X-Content-Type-Options option

	// server parts
	RootCAs       string   `json:"rootCAs"`      // server Root CAs path
	ServerCrt     string   `json:"server_cert"`  // server certificate
	ServerKey     string   `json:"server_key"`   // server certificate
	DomainNames   []string `json:"domain_names"` // LetsEncrypt domain names
	LimiterPeriod string   `json:"rate"`         // limiter rate value

	// MetaData parts
	DBURI      string     `json:"db_uri"`   // meta-data server URI
	DBName     string     `json:"db_name"`  // meta-data database name
	DBColl     string     `json:"db_coll"`  // meta-data database collection
	MLBackends MLBackends `json:"backends"` // ML backends

	// storage parts
	StorageDir string `json:"storage_dir"` // storage directory
}

// Config variable represents configuration object
var Config Configuration

// helper function to parse server configuration file
func parseConfig(configFile string) error {
	data, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		log.Println("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("Unable to parse", err)
		return err
	}

	// default values
	if Config.Port == 0 {
		Config.Port = 8181
	}
	if Config.LimiterPeriod == "" {
		Config.LimiterPeriod = "100-S"
	}
	if Config.MLBackends == nil {
		Config.MLBackends = make(MLBackends)
	}
	if Config.StaticDir == "" {
		cdir, err := os.Getwd()
		if err == nil {
			Config.StaticDir = fmt.Sprintf("%s/static", cdir)
		} else {
			Config.StaticDir = "static"
		}
	}
	if Config.StorageDir == "" {
		Config.StorageDir = "/tmp"
	}
	return nil
}
