package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/yanzay/log"
)

type contextKey string

var (
	addr = flag.String("addr", ":8080", "Address to listen")
)

type application struct {
	oauth         *oAuth
	indexResponse []byte
}

func main() {
	flag.Parse()
	app := &application{}

	oauth, err := newOAuth(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"))
	if err != nil {
		log.Fatalf("can't initialize github oauth: %v", err)
	}
	app.oauth = oauth

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(assetFS())))
	http.Handle("/favicon.ico", cachedHandler(MustAsset("static/favicon.ico")))
	http.Handle("/callback", app.oauth)
	http.HandleFunc("/", cachedHandler(MustAsset("static/index.html")))
	http.HandleFunc("/login", app.loginHandler)
	http.HandleFunc("/me", app.auth(app.meHandler))

	log.Infof("Starting server at %s", *addr)
	http.ListenAndServe(*addr, nil)
}

func (a *application) auth(f http.HandlerFunc) http.HandlerFunc {
	log.Trace("auth: construct")
	return func(w http.ResponseWriter, r *http.Request) {
		log.Trace("auth: start")
		h := r.Header.Get("Authorization")
		token, err := a.parseToken(h)
		if err != nil {
			log.Errorf("can't parse token: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), contextKey("email"), token.email)
		log.Trace("auth: invoke handler")
		f(w, r.WithContext(ctx))
	}
}

func cachedHandler(cache []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(cache)
	}
}

func (a *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("auth")
	if err != nil || c == nil {
		http.SetCookie(w, &http.Cookie{Name: "auth", Value: "", Expires: time.Now()})
		http.Redirect(w, r, a.oauth.url, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

func (a *application) meHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("meHandler: start")
	email, ok := r.Context().Value(contextKey("email")).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Tracef("meHandler: got email: %s", email)
	resp, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		http.Error(w, "oops", http.StatusInternalServerError)
		return
	}
	log.Tracef("meHandler: writing response")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

type userToken struct {
	email string
}

func (a *application) parseToken(tokenString string) (*userToken, error) {
	log.Trace("parseToken: start")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(a.oauth.config.ClientSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	email, ok := claims["email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	log.Trace("parseToken: return")
	return &userToken{
		email: email,
	}, nil
}
