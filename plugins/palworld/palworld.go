package palworld

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iBug/api-ustc/common"
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

func (c *Client) GetStatus() (Status, error) {
	status := Status{}
	msg, err := c.commander.Execute("ShowPlayers")
	if err != nil {
		return status, err
	}
	s := bufio.NewScanner(strings.NewReader(msg))
	s.Scan() // Discard first line
	for s.Scan() {
		line := s.Text()
		line = strings.Split(line, ",")[0]
		status.Players = append(status.Players, line)
	}
	status.Time = time.Now().Truncate(time.Second)
	return status, nil
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
	common.Services.Register("palworld", NewClient)
}
