package writefile

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/iBug/uniAPI/common"
)

type Service struct {
	File  string `json:"file"`
	Limit int64  `json:"limit"`
}

func (s *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	if req.Method != http.MethodPost {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	f, err := os.Create(s.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var n int64
	if s.Limit > 0 {
		n, err = io.CopyN(f, req.Body, s.Limit)
	} else {
		n, err = io.Copy(f, req.Body)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Leftover data from io.CopyN
	n, err = io.Copy(io.Discard, req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if n > 0 {
		http.Error(w, fmt.Sprintf("%d", n), http.StatusRequestEntityTooLarge)
		return
	}

	http.Error(w, fmt.Sprintf("%d", n), http.StatusOK)
}

func init() {
	common.Services.Register("writefile", common.NewService[*Service])
}
