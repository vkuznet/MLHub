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
	sessions "github.com/dghubble/sessions"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
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

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore[any](sessions.DebugCookieConfig, []byte(sessionSecret), nil)

const (
	sessionName     = "example-github-app"
	sessionSecret   = "example cookie signing secret"
	sessionUserKey  = "githubID"
	sessionUsername = "githubUsername"
)

// issueSession issues a cookie session after successful Github login
func issueSession() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		githubUser, err := github.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Set(sessionUserKey, *githubUser.ID)
		session.Set(sessionUsername, *githubUser.Login)
		if err := session.Save(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, req, "/profile", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

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
		RedirectURL:  fmt.Sprintf("http://localhost:%d%s/oauth/redirect", Config.Port, Config.Base),
		Endpoint:     githubOAuth2.Endpoint,
	}
	stateConfig := gologin.DebugOnlyCookieConfig
	fl := github.StateHandler(stateConfig, github.LoginHandler(config, nil))
	fc := github.StateHandler(stateConfig, github.CallbackHandler(config, issueSession(), nil))
	// fl, fc are type of http.Handler and we need to use HTTP router
	router.Router.GET(base+"/github/login", bunrouter.HTTPHandler(fl))
	router.Router.GET(base+"/github/callback", bunrouter.HTTPHandler(fc))
	router.GET(base+"/login", LoginHandler)
	router.GET(base+"/oauth/redirect", AccessHandler)

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
