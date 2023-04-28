package main

// utils module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"io"
	"compress/gzip"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

// RootCAs returns cert pool of our root CAs
func RootCAs() *x509.CertPool {
	log.Println("Load RootCAs from", Config.RootCAs)
	rootCAs := x509.NewCertPool()
	files, err := ioutil.ReadDir(Config.RootCAs)
	if err != nil {
		log.Printf("Unable to list files in '%s', error: %v\n", Config.RootCAs, err)
		return rootCAs
	}
	for _, finfo := range files {
		fname := fmt.Sprintf("%s/%s", Config.RootCAs, finfo.Name())
		caCert, err := os.ReadFile(filepath.Clean(fname))
		if err != nil {
			if Config.Verbose > 1 {
				log.Printf("Unable to read %s\n", fname)
			}
		}
		if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
			if Config.Verbose > 1 {
				log.Printf("invalid PEM format while importing trust-chain: %q", fname)
			}
		}
		if Config.Verbose > 1 {
			log.Println("Load CA file", fname)
		}
	}
	return rootCAs
}

// LetsEncryptServer provides HTTPs server with Let's encrypt for
// given domain names (hosts)
func LetsEncryptServer(hosts ...string) *http.Server {
	// setup LetsEncrypt cert manager
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Cache:      autocert.DirCache("certs"),
	}

	tlsConfig := &tls.Config{
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyPeerCertificate.
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequestClientCert,
		RootCAs:            RootCAs(),
		GetCertificate:     certManager.GetCertificate,
	}

	// start HTTP server with our rootCAs and LetsEncrypt certificates
	server := &http.Server{
		Addr:      ":https",
		TLSConfig: tlsConfig,
	}
	// start cert Manager goroutine
	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
	log.Println("Starting LetsEncrypt HTTPs server")
	return server
}

// LogName return proper log name based on Config.LogName and either
// hostname or pod name (used in k8s environment).
func LogName() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("unable to get hostname", err)
	}
	if os.Getenv("MY_POD_NAME") != "" {
		hostname = os.Getenv("MY_POD_NAME")
	}
	logName := Config.LogFile + "_%Y%m%d"
	if hostname != "" {
		logName = fmt.Sprintf("%s_%s", Config.LogFile, hostname) + "_%Y%m%d"
	}
	return logName
}

// ListEntry identifies types used by list's generics function
type ListEntry interface {
        int | int64 | float64 | string
}

// InList checks item in a list
func InList[T ListEntry](a T, list []T) bool {
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
}

// GzipReader struct to handle GZip'ed content of HTTP requests
type GzipReader struct {
        *gzip.Reader
        io.Closer
}

// Close helper function to close gzip reader
func (gz GzipReader) Close() error {
        return gz.Closer.Close()
}
