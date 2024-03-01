package robotstxt

import (
	"encoding/json"
	"net/http"

	"github.com/iBug/uniAPI/common"
)

type Service struct{}

func (Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User-Agent: *\nDisallow: /\n"))
}

func NewService(_ json.RawMessage) (common.Service, error) {
	return Service{}, nil
}

func init() {
	common.Services.Register("robotstxt", NewService)
}
