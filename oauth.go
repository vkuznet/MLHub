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

const (
	sessionName     = "MLHub-app"
	sessionSecret   = "signing secret"
	sessionUserKey  = "ID"
	sessionUsername = "Username"
	sessionToken    = "token"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore[any](sessions.DebugCookieConfig, []byte(sessionSecret), nil)

// issueSession issues a cookie session after successful provider login
func issueSession(provider string) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		session := sessionStore.New(sessionName)
		ctx := req.Context()
		log.Println("### issueSession", provider)
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
