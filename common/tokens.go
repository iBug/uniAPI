package common

import (
	"net/http"
	"strings"
)

func ValidateToken(header string, tokens []string) bool {
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

type TokenProtectedHandler struct {
	Next   http.Handler
	Tokens []string
}

// ServeHTTP implements the http.Handler interface.
func (h *TokenProtectedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("CF-Connecting-IP") != "" &&
		!ValidateToken(r.Header.Get("Authorization"), h.Tokens) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	h.Next.ServeHTTP(w, r)
}
