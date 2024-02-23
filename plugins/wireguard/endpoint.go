package wireguard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/iBug/api-ustc/common"
)

type Service struct {
	PublicKey string `json:"public-key"`
	Interface string `json:"interface"`
	UseSudo   bool   `json:"use-sudo"`
}

func stripPort(s string) string {
	lastIndex := strings.LastIndex(s, ":")
	if lastIndex == -1 {
		return s
	}
	return s[:lastIndex]
}

func (s *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	args := []string{"wg", "show", s.Interface, "endpoints"}
	if s.UseSudo {
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("internal server error: %v\n", err), http.StatusInternalServerError)
		return
	}

	cmd.Start()
	defer cmd.Wait()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			break
		}
		if parts[0] == s.PublicKey {
			ip := stripPort(parts[1])
			http.Error(w, ip, http.StatusOK)
			return
		}
	}
	http.Error(w, "server not found", http.StatusInternalServerError)
}

func NewService(config json.RawMessage) (common.Service, error) {
	s := new(Service)
	err := json.Unmarshal(config, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func init() {
	common.Services.Register("wireguard.endpoint", NewService)
}
