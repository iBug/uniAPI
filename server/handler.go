package server

import (
	"net/http"
	"sync"
)

type ReloadableHandler struct {
	mu sync.RWMutex
	h  http.Handler
}

func (h *ReloadableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Get().ServeHTTP(w, r)
}

func (h *ReloadableHandler) Get() http.Handler {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.h
}

func (h *ReloadableHandler) Set(handler http.Handler) {
	h.mu.Lock()
	h.h = handler
	h.mu.Unlock()
}
