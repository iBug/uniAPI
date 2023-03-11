package factorio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	rcon "github.com/forewing/csgo-rcon"
)

type FactorioStatus struct {
	Time    time.Time `json:"time"`
	Players []string  `json:"players"`
}

const (
	FAC_SERVER_ADDR = "10.255.0.9"
	FAC_RCON_PORT   = 34197
	FAC_RCON_PASS   = "ohmy206rcon"
)

var (
	facRcon = rcon.New(fmt.Sprintf("%s:%d", FAC_SERVER_ADDR, FAC_RCON_PORT), FAC_RCON_PASS, time.Millisecond*100)
)

func GetFactorioStatus() (status FactorioStatus, err error) {
	response, err := facRcon.Execute("/players online")
	if err != nil {
		return
	}
	s := bufio.NewScanner(strings.NewReader(response))
	s.Scan()
	for s.Scan() {
		text := s.Text()
		if !strings.HasPrefix(text, "  ") {
			continue
		}
		text = strings.TrimPrefix(text, "  ")
		text = strings.TrimSuffix(text, " (online)")
		status.Players = append(status.Players, text)
	}
	status.Time = time.Now().Truncate(time.Second)
	return
}

func Handle206Factorio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := GetFactorioStatus()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`))
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=5")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
