package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/uptrace/bunrouter"
)

// helper function to get base path
func basePath(s string) string {
	if Config.Base != "" {
		if strings.HasPrefix(s, "/") {
			s = strings.Replace(s, "/", "", 1)
		}
		if strings.HasPrefix(Config.Base, "/") {
			return fmt.Sprintf("%s/%s", Config.Base, s)
		}
		return fmt.Sprintf("/%s/%s", Config.Base, s)
	}
	return s
}

// http handlers based on gorilla/mux
func handlers() *mux.Router {
	router := mux.NewRouter()

	// visible routes
	router.HandleFunc(basePath("/model/{model:[a-zA-Z0-9_]+}/predict/image"), PredictHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/model/{model:[a-zA-Z0-9_]+}/predict"), PredictHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/model/{model:[a-zA-Z0-9_]+}/upload"), UploadHandler)
	router.HandleFunc(basePath("/upload"), UploadHandler)
	router.HandleFunc(basePath("/model/{model:[a-zA-Z0-9_]+}/download"), DownloadHandler)
	router.HandleFunc(basePath("/model/{model:[a-zA-Z0-9_]+}"), RequestHandler)
	router.HandleFunc(basePath("/models"), ModelsHandler).Methods("GET")
	router.HandleFunc(basePath("/status"), StatusHandler).Methods("GET")
	router.HandleFunc(basePath("/favicon.ico"), FaviconHandler).Methods("GET")
	router.HandleFunc(basePath("/"), RequestHandler).Methods("GET")

	// static handlers
	for _, dir := range []string{"js", "css", "images"} {
		m := fmt.Sprintf("%s/%s/", Config.Base, dir)
		d := fmt.Sprintf("%s/%s", Config.StaticDir, dir)
		hdlr := http.StripPrefix(m, http.FileServer(http.Dir(d)))
		http.Handle(m, hdlr)
	}

	// log all requests
	router.Use(loggingMiddleware)
	// use limiter middleware to slow down clients
	router.Use(limitMiddleware)

	return router
}

// bunrouter implementation of the compatible (with net/http) router handlers
func bunRouter() *bunrouter.CompatRouter {
	router := bunrouter.New(
		bunrouter.Use(bunrouterLoggingMiddleware),
		bunrouter.Use(bunrouterLimitMiddleware),
	).Compat()
	base := Config.Base
	router.GET(base+"/", RequestHandler)
	router.GET(base+"/favicon.ico", FaviconHandler)
	router.GET(base+"/status", StatusHandler)
	router.GET(base+"/models", ModelsHandler)
	router.GET(base+"/model/:model/predict/image", PredictHandler)
	router.POST(base+"/model/:model/predict/image", PredictHandler)
	router.GET(base+"/model/:model/predict", PredictHandler)
	router.POST(base+"/model/:model/predict", PredictHandler)
	router.POST(base+"/model/:model/upload", UploadHandler)
	router.GET(base+"/upload", UploadHandler)
	router.GET(base+"/model/:model/download", DownloadHandler)
	router.GET(base+"/model/:model", RequestHandler)

	// static handlers
	for _, dir := range []string{"js", "css", "images"} {
		m := fmt.Sprintf("%s/%s", Config.Base, dir)
		d := fmt.Sprintf("%s/%s", Config.StaticDir, dir)
		hdlr := http.StripPrefix(m, http.FileServer(http.Dir(d)))
		// invoke bunrouter from Compat to setup static content
		router.Router.GET(m+"/*path", bunrouter.HTTPHandler(hdlr))
	}
	return router
}

// Serve a reverse proxy for a given url
func reverseProxy(targetURL string, w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// parse the url
	url, _ := url.Parse(targetURL)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// set custom transport to capture size of response body
	//     proxy.Transport = &transport{http.DefaultTransport}
	if Config.Verbose > 2 {
		log.Printf("HTTP headers: %+v\n", r.Header)
	}

	// handle double slashes in request path
	r.URL.Path = strings.Replace(r.URL.Path, "//", "/", -1)

	// Update the headers to allow for SSL redirection
	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.URL.User = url.User
	if Config.Verbose > 0 {
		log.Printf("redirect to url.Scheme=%s url.User=%s url.Host=%s", r.URL.Scheme, r.URL.User, r.URL.Host)
	}
	if url.User != nil {
		// set basic authorization for provided user credentials
		hash := base64.StdEncoding.EncodeToString([]byte(url.User.String()))
		r.Header.Set("Authorization", fmt.Sprintf("Basic %s", hash))
	}
	reqHost := r.Header.Get("Host")
	if reqHost == "" {
		name, err := os.Hostname()
		if err == nil {
			reqHost = name
		}
	}

	// XForward headers
	if Config.XForwardedHost != "" {
		r.Header.Set("X-Forwarded-Host", Config.XForwardedHost)
	} else {
		r.Header.Set("X-Forwarded-Host", reqHost)
	}
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	r.Host = url.Host
	if Config.Verbose > 0 {
		log.Printf("proxy request: %+v\n", r)
	}

	// use custom modify response function to setup response headers
	proxy.ModifyResponse = func(resp *http.Response) error {
		if Config.Verbose > 0 {
			log.Println("proxy ModifyResponse")
		}
		if Config.XContentTypeOptions != "" {
			resp.Header.Set("X-Content-Type-Options", Config.XContentTypeOptions)
		}
		resp.Header.Set("Response-Status", resp.Status)
		resp.Header.Set("Response-Status-Code", fmt.Sprintf("%d", resp.StatusCode))
		resp.Header.Set("Response-Proto", resp.Proto)
		resp.Header.Set("Response-Time", time.Since(start).String())
		resp.Header.Set("Response-Time-Seconds", fmt.Sprintf("%v", time.Since(start).Seconds()))
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
		if Config.Verbose > 0 {
			log.Printf("proxy ErrorHandler error was: %+v", err)
		}
		header := rw.Header()
		header.Set("Response-Status", fmt.Sprintf("%d", http.StatusBadGateway))
		header.Set("Response-Status-Code", fmt.Sprintf("%d", http.StatusBadGateway))
		header.Set("Response-Time", time.Since(start).String())
		header.Set("Response-Time-Seconds", fmt.Sprintf("%v", time.Since(start).Seconds()))
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
	}

	// ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(w, r)
}

// Server implements MLaaS server
func Server() {

	initLimiter(Config.LimiterPeriod)
	// gorilla/mux handlers
	//     http.Handle(basePath("/"), handlers())

	// bunrouter implementation of the router
	router := bunRouter()

	// start HTTPs server
	if len(Config.DomainNames) > 0 {
		server := LetsEncryptServer(Config.DomainNames...)
		log.Println("Start HTTPs server with LetsEncrypt", Config.DomainNames)
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else if Config.ServerCrt != "" && Config.ServerKey != "" {
		tlsConfig := &tls.Config{
			RootCAs: RootCAs(),
		}
		server := &http.Server{
			Addr:      ":https",
			TLSConfig: tlsConfig,
			Handler:   router, // it is not used in gorilla/mux
		}
		log.Printf("Start HTTPs server with %s and %s on :%d", Config.ServerCrt, Config.ServerKey, Config.Port)
		log.Fatal(server.ListenAndServeTLS(Config.ServerCrt, Config.ServerKey))
	} else {
		log.Printf("Start HTTP server on :%d", Config.Port)
		// for gorilla/mux we do not pass router but rather use http module itself
		//         http.ListenAndServe(fmt.Sprintf(":%d", Config.Port), nil)

		// for bunrouter we pass router handler
		http.ListenAndServe(fmt.Sprintf(":%d", Config.Port), router)
	}
}
