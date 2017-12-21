package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"honnef.co/go/js/dom"
)

type app struct {
	token   string
	doc     dom.HTMLDocument
	window  dom.Window
	changes chan changeFunc
}

type changeFunc func(dom.HTMLDocument)

type user struct {
	Email string
}

func newApp() *app {
	window := dom.GetWindow()
	doc := window.Document().(dom.HTMLDocument)
	token := getCookie(doc, "auth")
	return &app{
		window:  window,
		doc:     doc,
		token:   token,
		changes: make(chan changeFunc, 100),
	}
}

func main() {
	a := newApp()
	a.render(0)
	me, err := a.fetchMe()
	if err != nil {
		fmt.Println(err)
		return
	}
	a.renderHeader(me)
}

func (a *app) renderHeader(u *user) {
	a.changes <- func(doc dom.HTMLDocument) {
		doc.GetElementByID("user-email").SetInnerHTML(u.Email)
	}
}

func (a *app) render(time.Duration) {
	for {
		select {
		case change := <-a.changes:
			change(a.doc)
		default:
			a.window.RequestAnimationFrame(a.render)
			return
		}
	}
}

func (a *app) fetchMe() (*user, error) {
	resp, err := a.request("/me")
	if err != nil {
		return nil, err
	}
	u := &user{}
	err = json.Unmarshal(resp, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (a *app) request(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", a.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getCookie(doc dom.HTMLDocument, name string) string {
	all := doc.Cookie()
	cookies := strings.Split(all, ";")
	for _, cookie := range cookies {
		pair := strings.Split(strings.TrimSpace(cookie), "=")
		if len(pair) > 1 && pair[0] == name {
			return pair[1]
		}
	}
	return ""
}
