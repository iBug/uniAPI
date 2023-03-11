package ustc

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net"
	"net/http"
	"net/url"
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

var (
	ustcTimeout = 3 * time.Second
	ustcDialer  = &net.Dialer{
		Timeout:   ustcTimeout,
		KeepAlive: ustcTimeout,
		LocalAddr: &net.TCPAddr{IP: net.ParseIP("10.255.1.3")},
	}
	ustcTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           ustcDialer.DialContext,
		MaxIdleConns:          3,
		IdleConnTimeout:       10 * ustcTimeout,
		TLSHandshakeTimeout:   ustcTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
	ustcClient = &http.Client{Transport: ustcTransport, Timeout: ustcTimeout}
)

func HandleUstcId(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		log.Print("Missing 'id' parameter")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest("GET", "https://api.lib.ustc.edu.cn/get_info_from_id.php", nil)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	q := url.Values{}
	q.Add("id", id)
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&info)
}
