package main

import (
	"log"
	"net/http"

	"github.com/dghubble/gologin/v2/github"
	google "github.com/dghubble/gologin/v2/google"
	oauth2Login "github.com/dghubble/gologin/v2/oauth2"
	twitter "github.com/dghubble/gologin/v2/twitter"
	sessions "github.com/dghubble/sessions"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore[any](sessions.DebugCookieConfig, []byte(sessionSecret), nil)

const (
	sessionName     = "MLHub-app"
	sessionSecret   = "signing secret"
	sessionUserKey  = "ID"
	sessionUsername = "Username"
	sessionToken    = "token"
)

// issueSession issues a cookie session after successful Github login
func issueSession(provider string) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		session := sessionStore.New(sessionName)
		ctx := req.Context()
		if token, err := oauth2Login.TokenFromContext(ctx); err == nil {
			session.Set(sessionToken, token.AccessToken)
		} else {
			log.Println("ERROR: fail to obtain OAuth2 token", err)
		}
		if provider == "github" {
			if user, err := github.UserFromContext(ctx); err == nil {
				session.Set(sessionUserKey, *user.ID)
				session.Set(sessionUsername, *user.Login)
			} else {
				log.Println("ERROR: fail to obtain github credentials", err)
			}
		} else if provider == "google" {
			if user, err := google.UserFromContext(ctx); err == nil {
				session.Set(sessionUserKey, user.Id)
				session.Set(sessionUsername, user.Name)
			} else {
				log.Println("ERROR: fail to obtain google credentials", err)
			}
		} else if provider == "twitter" {
			if user, err := twitter.UserFromContext(ctx); err == nil {
				session.Set(sessionUserKey, user.ID)
				session.Set(sessionUsername, user.ScreenName)
			} else {
				log.Println("ERROR: fail to obtain twitter credentials", err)
			}
		}
		if err := session.Save(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, req, "/access", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

// googleSession googles a cookie session after successful Google login
func googleSession() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		user, err := google.UserFromContext(ctx)
		token, _ := oauth2Login.TokenFromContext(ctx)
		if Config.Verbose > 0 {
			log.Printf("googleSession\nTOKEN: %+v\nUSER: %+v", token, user)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Set(sessionUserKey, user.Id)
		session.Set(sessionUsername, user.Name)
		session.Set(sessionToken, token.AccessToken)
		if err := session.Save(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, req, "/access", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}
