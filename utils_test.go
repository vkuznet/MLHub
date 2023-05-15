package main

import (
	"log"
)

// helper function to initialize MetaData service
func initMetaDataService() {
	// initialize schema manager which holds our schemas
	configFile := "config-test.json"
	parseConfig(configFile)
	// use verbose log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
