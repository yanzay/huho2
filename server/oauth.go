package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/yanzay/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	githubEmailsURL = "https://api.github.com/user/emails"
)

type oAuth struct {
	config *oauth2.Config
	url    string
}

func newOAuth(clientID, clientSecret string) (*oAuth, error) {
	if clientID == "" {
		return nil, fmt.Errorf("github client ID required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("github client secret required")
	}
	oa := &oAuth{}
	oa.config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	oa.url = oa.config.AuthCodeURL("state", oauth2.AccessTypeOnline)
	return oa, nil
}

func (oa *oAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	tok, err := oa.config.Exchange(ctx, code)
	if err != nil {
		log.Error(err)
		http.Error(w, "auth error", http.StatusBadRequest)
		return
	}
	if !tok.Valid() {
		log.Errorf("received token is invalid")
		http.Error(w, "invalid auth token", http.StatusBadRequest)
		return
	}
	client := oa.config.Client(ctx, tok)
	resp, err := client.Get(githubEmailsURL)
	if err != nil {
		log.Error(err)
		return
	}
	emails := []*struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&emails)
	if err != nil {
		log.Error(err)
		return
	}
	for _, email := range emails {
		if email.Primary {
			log.Infof("found primary user email: %s", email.Email)
			jtoken, err := oa.createJWT(email.Email, tok)
			if err != nil {
				log.Errorf("unable to create JWT for %s: %v", email.Email, err)
				http.Error(w, "oops", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "auth", Value: jtoken})
			http.Redirect(w, r, "/", 302)
		}
	}
}

func (oa *oAuth) createJWT(email string, tok *oauth2.Token) (string, error) {
	log.Debugf("creating token: %s", email)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
	})
	return token.SignedString([]byte(oa.config.ClientSecret))
}
