package minecraft

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/iBug/api-ustc/common"
)

type Status struct {
	Time     time.Time `json:"time"`
	Count    int       `json:"count"`
	MaxCount int       `json:"max_count"`
	Players  []string  `json:"players"`
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

var RePlayerList = *regexp.MustCompile(`^There are (\d+) of a max of (\d+) players online: `)

func (c *Client) GetStatus() (Status, error) {
	status := Status{}
	msg, err := c.commander.Execute("list")
	if err != nil {
		return status, err
	}
	m := RePlayerList.FindStringSubmatch(msg)
	status.Count, _ = strconv.Atoi(m[1])
	status.MaxCount, _ = strconv.Atoi(m[2])
	playersStr := strings.SplitN(msg, ": ", 2)[1]
	if len(playersStr) > 0 {
		status.Players = strings.Split(strings.SplitN(msg, ": ", 2)[1], ", ")
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
	common.Services.Register("minecraft", NewClient)
}
