package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"

	gologin "github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/github"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"

	google "github.com/dghubble/gologin/v2/google"
	googleOAuth2 "golang.org/x/oauth2/google"
)

// metadata represents MetaData instance
var metadata *MetaData

// content is our static web server content.
//go:embed static
var StaticFs embed.FS

// The OAuth parts are based on
// https://github.com/dghubble/gologin
// package where we explid github authentication, see
// https://github.com/dghubble/gologin/blob/main/examples/github

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

// bunrouter implementation of the compatible (with net/http) router handlers
func bunRouter() *bunrouter.CompatRouter {
	router := bunrouter.New(
		bunrouter.Use(bunrouterLoggingMiddleware),
		bunrouter.Use(bunrouterLimitMiddleware),
	).Compat()
	base := Config.Base
	router.GET(base+"/", RequestHandler)
	router.GET(base+"/favicon.ico", FaviconHandler)

	// model APIs
	router.GET(base+"/model/:model/predict/image", PredictHandler)
	router.POST(base+"/model/:model/predict/image", PredictHandler)
	router.GET(base+"/model/:model/predict", PredictHandler)
	router.POST(base+"/model/:model/predict", PredictHandler)
	router.POST(base+"/model/:model/upload", UploadHandler)
	router.GET(base+"/model/:model/download", DownloadHandler)
	router.GET(base+"/model/:model", RequestHandler)

	// web APIs
	router.GET(base+"/status", StatusHandler)
	router.GET(base+"/docs", DocsHandler)
	router.GET(base+"/models", ModelsHandler)
	router.GET(base+"/upload", UploadHandler)
	router.GET(base+"/domains", DomainsHandler)
	router.GET(base+"/download", DownloadHandler)
	router.GET(base+"/inference", InferenceHandler)
	// POST APIs
	router.POST(base+"/upload", UploadHandler)
	router.POST(base+"/predict", PredictHandler)
	router.POST(base+"/download", DownloadHandler)

	// auth end-points
	config := &oauth2.Config{
		ClientID:     Config.ClientID,
		ClientSecret: Config.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://localhost:%d%s/github/callback", Config.Port, Config.Base),
		Endpoint:     githubOAuth2.Endpoint,
	}
	stateConfig := gologin.DebugOnlyCookieConfig
	githubLogin := github.StateHandler(stateConfig, github.LoginHandler(config, nil))
	githubCallback := github.StateHandler(stateConfig, github.CallbackHandler(config, githubSession(), nil))
	router.Router.GET(base+"/github/login", bunrouter.HTTPHandler(githubLogin))
	router.Router.GET(base+"/github/callback", bunrouter.HTTPHandler(githubCallback))

	config = &oauth2.Config{
		ClientID:     Config.ClientID,
		ClientSecret: Config.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://localhost:%d%s/google/callback", Config.Port, Config.Base),
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}
	googleLogin := google.StateHandler(stateConfig, google.LoginHandler(config, nil))
	googleCallback := google.StateHandler(stateConfig, google.CallbackHandler(config, googleSession(), nil))
	router.Router.GET(base+"/google/login", bunrouter.HTTPHandler(googleLogin))
	router.Router.GET(base+"/google/callback", bunrouter.HTTPHandler(googleCallback))

	router.GET(base+"/login", LoginHandler)
	router.GET(base+"/access", AccessHandler)
	//     router.GET(base+"/oauth/redirect", AccessHandler)

	// static handlers
	for _, dir := range []string{"js", "css", "images"} {
		filesFS, err := fs.Sub(StaticFs, "static/"+dir)
		if err != nil {
			panic(err)
		}
		m := fmt.Sprintf("%s/%s", Config.Base, dir)
		fileServer := http.FileServer(http.FS(filesFS))
		hdlr := http.StripPrefix(m, fileServer)
		router.Router.GET(m+"/*path", bunrouter.HTTPHandler(hdlr))
	}

	// static model download area
	bpath := fmt.Sprintf("%s/bundles", base)
	hdlr := http.StripPrefix(bpath, http.FileServer(http.Dir(Config.StorageDir)))
	router.Router.GET(base+"/bundles/*path", bunrouter.HTTPHandler(hdlr))

	return router
}

// Server implements MLaaS server
func Server() {

	// initialize server middleware
	initLimiter(Config.LimiterPeriod)

	// initialize metadata
	metadata = &MetaData{DBName: Config.DBName, DBColl: Config.DBColl}

	// setup server router
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
			Handler:   router,
		}
		log.Printf("Start HTTPs server with %s and %s on :%d", Config.ServerCrt, Config.ServerKey, Config.Port)
		log.Fatal(server.ListenAndServeTLS(Config.ServerCrt, Config.ServerKey))
	} else {
		log.Printf("Start HTTP server on :%d", Config.Port)
		http.ListenAndServe(fmt.Sprintf(":%d", Config.Port), router)
	}
}
