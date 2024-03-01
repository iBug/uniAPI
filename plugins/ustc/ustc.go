package ustc

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/iBug/uniAPI/common"
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

type UstcIdConfig struct {
	BindAddress string `json:"bind-address"`
	Timeout     string `json:"timeout"`
}

type UstcIdService struct {
	client *http.Client
}

func (s *UstcIdService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	res, err := s.client.Do(req)
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
		log.Printf("XML decode error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&info)
}

func NewService(rawConfig json.RawMessage) (common.Service, error) {
	var config UstcIdConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	httpTimeout := common.ParseDurationDefault(config.Timeout, 3*time.Second)
	httpDialer := &net.Dialer{
		Timeout:   httpTimeout,
		KeepAlive: httpTimeout,
	}
	if config.BindAddress != "" {
		httpDialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(config.BindAddress)}
	}
	httpTransport := &http.Transport{
		DialContext:           httpDialer.DialContext,
		MaxIdleConns:          3,
		IdleConnTimeout:       10 * httpTimeout,
		TLSHandshakeTimeout:   httpTimeout,
		ExpectContinueTimeout: httpTimeout / 2,
	}
	httpClient := &http.Client{Transport: httpTransport, Timeout: httpTimeout}
	return &UstcIdService{client: httpClient}, nil
}

func init() {
	common.Services.Register("ustc-id", NewService)
}
