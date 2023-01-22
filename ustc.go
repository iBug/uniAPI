package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ReaderInfo struct {
	XMLName xml.Name `xml:"reader_info" json:"-"`
	IP      string   `xml:"ip" json:"-"`
	Status  string   `xml:"status" json:"status"`
	UserId  string   `xml:"salaryno" json:"user_id"`
	Name    string   `xml:"name" json:"name"`
	Type    string   `xml:"type" json:"type"`
	Email   string   `xml:"email" json:"email"`
}

func ValidateToken(header string, tokens []string) bool {
	parts := strings.Fields(header)
	if len(parts) > 2 {
		return false
	}
	token := parts[0]
	switch strings.ToLower(parts[0]) {
	case "bearer", "token":
		token = parts[1]
	}
	for _, t := range tokens {
		if token == t {
			return true
		}
	}
	return false
}

var (
	ustcDialer = &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 3 * time.Second,
		LocalAddr: &net.TCPAddr{IP: net.ParseIP("10.255.1.3")},
	}
	ustcTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           ustcDialer.DialContext,
		MaxIdleConns:          3,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	ustcClient = &http.Client{Transport: ustcTransport, Timeout: 3 * time.Second}
)

func HandleUstcId(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("CF-Connecting-IP") != "" &&
		!ValidateToken(r.Header.Get("Authorization"), config.UstcTokens) {
		w.WriteHeader(http.StatusForbidden)
	}

	req, err := http.NewRequest("GET", "https://api.lib.ustc.edu.cn/get_info_from_id.php", nil)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	q := url.Values{}
	q.Add("id", "SA21011003")
	req.URL.RawQuery = q.Encode()

	res, err := ustcClient.Do(req)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("USTC Library API returned %d", res.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var info ReaderInfo
	err = xml.NewDecoder(res.Body).Decode(&info)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&info)
}
