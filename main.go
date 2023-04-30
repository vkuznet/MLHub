package main

// mlaas-proxy - Go implementation of reverse proxy server for MLaaS backends
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/
)

// version of the code
var version string

// helper function to return version string of the server
func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("auth-proxy-server git=%s go=%s date=%s", version, goVersion, tstamp)
}

func main() {
	var config string
	flag.StringVar(&config, "config", "", "configuration file")
	var version bool
	flag.BoolVar(&version, "version", false, "print version information about the server")
	flag.Parse()
	if version {
		fmt.Println(info())
		os.Exit(0)
	}
	err := parseConfig(config)
	if err != nil {
		log.Fatalf("unable to parse config %s, error %v\n", config, err)
	}

	// configure logger with log time, filename, and line number
	log.SetFlags(0)
	if Config.Verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if Config.Verbose > 0 {
		log.Printf("%+v\n", Config)
	}

	// start proxy server
	Server()
}
