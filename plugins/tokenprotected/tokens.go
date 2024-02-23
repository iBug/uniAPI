package common

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/iBug/api-ustc/common"
)

func validateToken(header string, tokens []string) bool {
	parts := strings.Fields(header)
	if len(parts) == 0 || len(parts) > 2 {
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

type TokenProtectedConfig struct {
	Tokens  []string        `json:"tokens"`
	Service json.RawMessage `json:"service"`
}

type TokenProtectedService struct {
	next   common.Service
	tokens []string
}

// ServeHTTP implements the http.Handler interface.
func (s *TokenProtectedService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("CF-Connecting-IP") != "" &&
		!validateToken(r.Header.Get("Authorization"), s.tokens) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	s.next.ServeHTTP(w, r)
}

func NewTokenProtectedService(rawConfig json.RawMessage) (common.Service, error) {
	var config TokenProtectedConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}
	next, err := common.Services.NewFromConfig(config.Service)
	if err != nil {
		return nil, err
	}
	return &TokenProtectedService{
		next:   next,
		tokens: config.Tokens,
	}, nil
}

func init() {
	common.Services.Register("token-protected", NewTokenProtectedService)
}
