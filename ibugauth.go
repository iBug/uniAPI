package main

import (
	"net/http"
	"net/url"
)

func HandleIBugAuth(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ticket, ok := query["ticket"]
	if !ok {
		url, _ := url.ParseRequestURI("https://vlab.ustc.edu.cn/ibug-login/")
		q := url.Query()
		q.Add("host", r.Host)
		url.RawQuery = q.Encode()
		http.Redirect(w, r, url.String(), http.StatusFound)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "ibugauth",
		Value:    ticket[0],
		MaxAge:   172800,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
