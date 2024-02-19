package ibugauth

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iBug/api-ustc/common"
)

const (
	CasService     = "https://vlab.ustc.edu.cn/ibug-login/"
	CasValidateUrl = "https://passport.ustc.edu.cn/serviceValidate"
)

type CasAttributes struct {
	Xbm       int       `xml:"xbm"`
	LoginTime time.Time `xml:"logintime"`
	Gid       int       `xml:"gid"`
	Ryzxztdm  int       `xml:"ryzxztdm"`
	Ryfldm    int       `xml:"ryfldm"`
	LoginIP   string    `xml:"loginip"`
	Name      string    `xml:"name"`
	Login     string    `xml:"login"`
	Zjhm      string    `xml:"zjhm"`
	Glzjh     []string  `xml:"glzjh"`
	DeptCode  string    `xml:"deptCode"`
	Email     string    `xml:"email"`
}

type casAttributesA CasAttributes
type casAttributesS struct {
	*casAttributesA
	LoginTime string `xml:"logintime"`
	Glzjh     string `xml:"glzjh"`
}

func (c *CasAttributes) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	s := &casAttributesS{casAttributesA: (*casAttributesA)(c)}
	if err = d.DecodeElement(s, &start); err != nil {
		return
	}
	c.LoginTime, err = time.Parse("2006-01-02 15:04:05", s.LoginTime)
	c.Glzjh = strings.Fields(s.Glzjh)
	return
}

type CasInfo struct {
	XMLName               xml.Name
	AuthenticationSuccess *struct {
		User       string        `xml:"user"`
		Attributes CasAttributes `xml:"attributes"`
	} `xml:"authenticationSuccess" json:",omitempty"`
	AuthenticationFailure *struct {
		Code string `xml:"code,attr"`
		Data string `xml:",chardata"`
	} `xml:"authenticationFailure" json:",omitempty"`
}

func ParseCasInfo(r io.Reader) (*CasInfo, error) {
	data := &CasInfo{}
	dec := xml.NewDecoder(r)
	dec.DefaultSpace = "cas"
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

var stubCasInfo CasInfo

func ValidateCasTicket(ticket string) (*CasInfo, error) {
	if ticket == "x" {
		// debug ticket
		return &stubCasInfo, nil
	}
	url, _ := url.ParseRequestURI(CasValidateUrl)
	q := url.Query()
	q.Add("service", CasService)
	q.Add("ticket", ticket)
	url.RawQuery = q.Encode()
	resp, err := http.Get(url.String())
	if err != nil {
		log.Println(err)
		return &stubCasInfo, err
	}
	defer resp.Body.Close()
	return ParseCasInfo(resp.Body)
}

type Service struct{}

func (Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ticket, ok := query["ticket"]
	if !ok {
		url, _ := url.ParseRequestURI(CasService)
		q := url.Query()
		q.Add("host", r.Host)
		url.RawQuery = q.Encode()
		http.Redirect(w, r, url.String(), http.StatusFound)
	}

	info, err := ValidateCasTicket(ticket[0])
	if err != nil {
		log.Printf("Validate CAS ticket error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if info.AuthenticationSuccess != nil {
		res := info.AuthenticationSuccess
		log.Printf("CAS login by %s (%s) from %q\n", res.User, res.Attributes.Name, res.Attributes.LoginIP)
	} else if info.AuthenticationFailure != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "ibugauth",
		Value:    ticket[0],
		MaxAge:   172800,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func NewService(_ json.RawMessage) (common.Service, error) {
	return Service{}, nil
}

func init() {
	common.RegisterService("ibug-auth", NewService)
}
