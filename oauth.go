package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
	sessionProvider = "MLHub-Provider"
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
		// by default we will redirect to access end-point
		rpath := "/access"
		if req.URL != nil {
			// but if we get redirect query parameter we'll use it
			// to change redirect path
			redirect := req.URL.Query().Get("redirect")
			if redirect != "" {
				rpath = redirect
			}
		}
		if Config.Verbose > 0 {
			log.Printf("session redirect to '%s', request %+v", rpath, req)
		}
		http.Redirect(w, req, rpath, http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

/*
To check github token we can use the following API call:
curl -v -H "Authorization: Bearer $token" https://api.github.com/user
it will return something like this:
{
  "login": "UserName",
  "id": UserID,
  "type": "User",
  "name": "First Last name",
  "company": "Company Name",
  "location": "City, State",
  "bio": "Title associated with user",
}
*/

// UserData represents meta-data information about user
type UserData struct {
	Login    string
	ID       int
	Name     string
	Company  string
	Location string
	Bio      string
}

// helper function to get user data info
func githubTokenInfo(token string) (UserData, error) {
	var userData UserData
	// make HTTP call to github
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	uri := "https://api.github.com/user"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return userData, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	rsp, err := client.Do(req)
	if err != nil {
		return userData, err
	}
	defer rsp.Body.Close()
	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return userData, err
	}
	err = json.Unmarshal(data, &userData)
	return userData, err
}

// helper function to get user data info
func tokenInfo(token string, w http.ResponseWriter, r *http.Request) (*sessions.Session[any], error) {
	var userData UserData
	var err error
	var provider string
	providers := []string{"github", "google", "facebook", "twitter"}
	for _, p := range providers {
		if p == "github" {
			userData, err = githubTokenInfo(token)
		} else if p == "google" {
			err = errors.New(fmt.Sprintf("tokenInfo for p %s is not yet implemented", p))
		} else if p == "facebook" {
			err = errors.New(fmt.Sprintf("tokenInfo for p %s is not yet implemented", p))
		} else if p == "twitter" {
			err = errors.New(fmt.Sprintf("tokenInfo for p %s is not yet implemented", p))
		} else {
			err = errors.New(fmt.Sprintf("tokenInfo for p %s is not yet implemented", p))
		}
		if err == nil {
			provider = p
			break
		}
	}
	if token == "" {
		return nil, errors.New("No valid access token is provided")
	}
	if err != nil {
		msg := fmt.Sprintf("None of the existing providers %v can validate your token", providers)
		return nil, errors.New(msg)
	}
	userData, err = githubTokenInfo(token)
	if err != nil {
		return nil, err
	}
	if userData.Login == "" || userData.ID == 0 {
		return nil, errors.New("No valid user data is validated from" + provider)
	}
	// now if token is valid we will setup appropriate session cookies
	session, err := sessionStore.Get(r, sessionName)
	if err != nil {
		if r.Header.Get("Accept") == "application/json" {
			if Config.Verbose > 0 {
				log.Println("### tokenInfo create new session for HTTP CLI", r.Header.Get("User-Agent"), sessionName)
			}
			session = sessionStore.New(sessionName)
		} else {
			return session, err
		}
	}
	session.Set(sessionProvider, provider)
	session.Set(sessionToken, token)
	session.Set(sessionUserID, userData.ID)
	session.Set(sessionUserName, userData.Login)
	if err := session.Save(w); err != nil {
		log.Println("### tokenInfo saession saved error", err)
		return nil, err
	}
	log.Printf("### tokenInfo session is saved, %+v", session)
	return session, nil
}
