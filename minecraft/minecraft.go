package minecraft

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	rcon "github.com/forewing/csgo-rcon"
	"github.com/iBug/api-ustc/common"
)

type Status struct {
	Time     time.Time `json:"time"`
	Count    int       `json:"count"`
	MaxCount int       `json:"max_count"`
	Players  []string  `json:"players"`
}

type Config struct {
	common.RconConfig
}

type Client struct {
	ServerAddr string
	ServerPort int
	Password   string

	rcon *rcon.Client
}

var RE_MC_LIST = *regexp.MustCompile(`^There are (\d+) of a max of (\d+) players online: `)

func NewClient(serverAddr string, serverPort int, password string, timeout time.Duration) *Client {
	c := &Client{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		Password:   password,
	}
	c.rcon = rcon.New(fmt.Sprintf("%s:%d", c.ServerAddr, c.ServerPort), c.Password, timeout)
	return c
}

func (c *Client) GetStatus() (Status, error) {
	status := Status{}
	msg, err := c.rcon.Execute("list")
	if err != nil {
		return status, err
	}
	m := RE_MC_LIST.FindStringSubmatch(msg)
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
