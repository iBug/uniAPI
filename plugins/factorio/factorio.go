package factorio

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iBug/uniAPI/common"
)

type Status struct {
	Time    time.Time `json:"time"`
	Players []string  `json:"players"`
}

type Config struct {
	common.CommanderConfig
}

type Client struct {
	commander common.Commander
}

func NewClient(rawConfig json.RawMessage) (common.Service, error) {
	var config Config
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	commander, err := common.Commanders.NewFromConfig(config.Commander)
	if err != nil {
		return nil, err
	}
	return &Client{commander}, nil
}

func (c *Client) GetStatus() (status Status, err error) {
	response, err := c.commander.Execute("/players online")
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

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := c.GetStatus()
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

func init() {
	common.Services.Register("factorio", NewClient)
}
