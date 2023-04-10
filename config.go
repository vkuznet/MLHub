package main

// config module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

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
	return nil
}
