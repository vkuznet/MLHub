package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dghubble/gologin/v2/github"
	google "github.com/dghubble/gologin/v2/google"
	oauth2Login "github.com/dghubble/gologin/v2/oauth2"
	twitter "github.com/dghubble/gologin/v2/twitter"
	sessions "github.com/dghubble/sessions"
)

const (
	// here we keep names of cookies in our OAuth session
	sessionName     = "MLHub-App"
	sessionSecret   = "MLHub-Secret"
	sessionUserID   = "MLHub-UserID"
	sessionUserName = "MLHub-UserName"
	sessionToken    = "MLHub-Token"
	sessionProvider = "MLHun-Provider"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore[any](sessions.DebugCookieConfig, []byte(sessionSecret), nil)

// issueSession issues a cookie session after successful provider login
func issueSession(provider string) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		var userName, userID, token string
		session := sessionStore.New(sessionName)
		ctx := req.Context()
		if t, err := oauth2Login.TokenFromContext(ctx); err == nil {
			token = t.AccessToken
		} else {
			log.Println("ERROR: fail to obtain OAuth2 token", err)
		}
		if provider == "github" {
			if user, err := github.UserFromContext(ctx); err == nil {
				userID = fmt.Sprintf("%v", *user.ID)
				userName = fmt.Sprintf("%v", *user.Login)
			} else {
				log.Println("ERROR: fail to obtain github credentials", err)
			}
		} else if provider == "google" {
			if user, err := google.UserFromContext(ctx); err == nil {
				userID = fmt.Sprintf("%v", user.Id)
				userName = fmt.Sprintf("%v", user.Name)
			} else {
				log.Println("ERROR: fail to obtain google credentials", err)
			}
		} else if provider == "twitter" {
			if user, err := twitter.UserFromContext(ctx); err == nil {
				userID = fmt.Sprintf("%v", user.ID)
				userName = fmt.Sprintf("%v", user.ScreenName)
			} else {
				log.Println("ERROR: fail to obtain twitter credentials", err)
			}
		}
		session.Set(sessionProvider, provider)
		session.Set(sessionToken, token)
		session.Set(sessionUserID, userID)
		session.Set(sessionUserName, userName)
		if Config.Verbose > 0 {
			log.Printf("OAuth: provider %s user %s userID %s token %s", provider, userName, userID, token)
		}
		if err := session.Save(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, req, "/access", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}
