package csgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rcon "github.com/forewing/csgo-rcon"
	"github.com/iBug/api-ustc/common"
)

type LocalState struct {
	CTScore      int `json:"ct_score"`
	TScore       int `json:"t_score"`
	Map          string
	RoundsPlayed int  `json:"rounds_played"`
	GameOngoing  bool `json:"game_ongoing"`
}

type Status struct {
	Time        time.Time `json:"time"`
	Map         string    `json:"map"`
	GameMode    string    `json:"game_mode"`
	PlayerCount int       `json:"player_count"`
	BotCount    int       `json:"bot_count"`
	Players     []string  `json:"players"`

	LocalState LocalState `json:"local_state"`

	cvar_GameMode int
	cvar_GameType int
}

type OnlinePayload struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

var RE_PLAYERS = regexp.MustCompile(`(\d+) humans?, (\d+) bots?`)
var RE_CONNECTED = regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>" connected,`)
var RE_DISCONNECTED = regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>" disconnected \(`)
var RE_MATCH_STATUS = regexp.MustCompile(`MatchStatus: Score: (\d+):(\d+) on map "(\w+)" RoundsPlayed: (\d+)`)
var RE_GAME_OVER = regexp.MustCompile(`^(Game Over:)`)
var GAME_MODE_S = map[int]string{
	0:   "casual",
	1:   "competitive",
	2:   "scrim competitive",
	3:   "5v5 scrim competitive",
	100: "arms race",
	101: "demolition",
	102: "deathmatch",
	200: "training",
	300: "custom",
	400: "cooperative",
	500: "skirmish",
	600: "danger zone",
}

type Config struct {
	common.RconConfig
	Api         string `json:"api"`
	DisableFile string `json:"disable-file"`
}

type Client struct {
	ServerAddr string
	ServerPort int
	Password   string
	Timeout    time.Duration
	Api        string
	CacheTime  time.Duration

	SilentFunc func() bool

	savedStatus  Status
	localState   LocalState
	localStateMu sync.Mutex
	rcon         *rcon.Client
}

func NewClient(serverAddr string, serverPort int, password string, timeout time.Duration) *Client {
	c := &Client{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		Password:   password,
		Timeout:    timeout,
		CacheTime:  10 * time.Second,
	}
	c.Init()
	return c
}

func (c *Client) Init() {
	c.rcon = rcon.New(fmt.Sprintf("%s:%d", c.ServerAddr, c.ServerPort), c.Password, c.Timeout)
}

func (s *Status) ParseGameMode() string {
	// Source: https://totalcsgo.com/command/gamemode
	id := s.cvar_GameType*100 + s.cvar_GameMode
	if str, ok := GAME_MODE_S[id]; ok {
		return str
	}
	return "unknown"
}

func (c *Client) GetStatus(useCache bool) (Status, error) {
	if useCache && time.Since(c.savedStatus.Time) < c.CacheTime {
		return c.savedStatus, nil
	}

	msg, err := c.rcon.Execute("cvarlist game_; status")
	retries := 0
	for err != nil {
		retries++
		log.Printf("GetCsgoStatus rcon error %d: %v", retries, err)
		if retries >= 3 {
			return Status{}, fmt.Errorf("GetCsgoStatus error: %w", err)
		}
		time.Sleep(1 * time.Second)
		msg, err = c.rcon.Execute("cvarlist game_; status")
	}

	status := Status{Players: make([]string, 0, 10)}
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			items := strings.SplitN(line, ": ", 3)
			if len(items) < 2 {
				continue
			}
			key := strings.TrimSpace(items[0])
			value := strings.TrimSpace(items[1])
			switch key {
			case "map":
				status.Map = value
			case "players":
				matches := RE_PLAYERS.FindStringSubmatch(value)
				status.PlayerCount, _ = strconv.Atoi(matches[1])
				status.BotCount, _ = strconv.Atoi(matches[2])
			case "game_mode":
				status.cvar_GameMode, _ = strconv.Atoi(value)
			case "game_type":
				status.cvar_GameType, _ = strconv.Atoi(value)
			}
		} else {
			parts := strings.SplitN(line, "\"", 3)
			if len(parts) != 3 {
				continue
			}
			moreInfo := strings.Split(strings.TrimSpace(parts[2]), " ")
			if moreInfo[0] == "BOT" {
				continue
			}
			status.Players = append(status.Players, parts[1])
		}
	}
	status.GameMode = status.ParseGameMode()

	c.localStateMu.Lock()
	status.LocalState = c.localState
	c.localStateMu.Unlock()

	status.Time = time.Now().Truncate(time.Second)
	c.savedStatus = status
	return status, nil
}

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := c.GetStatus(true)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": err.Error()})
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=5")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (c *Client) ShouldSuppressNotice() bool {
	if c.SilentFunc != nil {
		return c.SilentFunc()
	}
	return false
}

func (c *Client) SendOnlineNotice(action, name string, count int) error {
	if c.ShouldSuppressNotice() {
		return nil
	}
	payloadObj := OnlinePayload{Action: action, Name: name, Count: count}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.Api, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "ping")
	res, err := http.DefaultClient.Do(req)
	retry := 0
	for err != nil {
		log.Printf("SendOnlineNotice error %d: %v\n", retry, err)
		retry++
		if retry >= 3 {
			return err
		}
		time.Sleep(1 * time.Second)
		res, err = http.DefaultClient.Do(req)
	}
	defer res.Body.Close()
	return nil
}

func (c *Client) HandleLogMessage(s string) {
	// Check online
	matches := RE_CONNECTED.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("%v connected\n", matches[1])
		status, err := c.GetStatus(false)
		if err != nil {
			log.Print(err)
			return
		}
		if status.PlayerCount < 1 || status.PlayerCount > 2 {
			return
		}
		err = c.SendOnlineNotice("goonline", matches[1], status.PlayerCount)
		if err != nil {
			log.Print(err)
		}
		return
	}

	// Check offline
	matches = RE_DISCONNECTED.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("%v disconnected\n", matches[1])
		status, err := c.GetStatus(false)
		if err != nil {
			log.Print(err)
			return
		}
		if status.PlayerCount > 0 {
			return
		}
		err = c.SendOnlineNotice("gooffline", matches[1], status.PlayerCount)
		if err != nil {
			log.Print(err)
		}
		return
	}

	// Check game state
	matches = RE_MATCH_STATUS.FindStringSubmatch(s)
	if len(matches) >= 4 {
		c.localStateMu.Lock()
		c.localState.CTScore, _ = strconv.Atoi(matches[1])
		c.localState.TScore, _ = strconv.Atoi(matches[2])
		c.localState.Map = matches[3]
		c.localState.RoundsPlayed, _ = strconv.Atoi(matches[4])
		c.localState.GameOngoing = c.localState.RoundsPlayed >= 0
		c.localStateMu.Unlock()
		return
	}

	// Check game over
	matches = RE_GAME_OVER.FindStringSubmatch(s)
	if len(matches) > 0 {
		c.localStateMu.Lock()
		c.localState.GameOngoing = false
		c.localStateMu.Unlock()
		return
	}
}

func (c *Client) OnlineWorker(ch <-chan string) {
	for s := range ch {
		c.HandleLogMessage(s)
	}
}

func (c *Client) LogServer(listenAddr string) error {
	serverAddr := net.ParseIP(c.ServerAddr)
	listenUDPAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("ResolveUDPAddr %#v: %w", listenAddr, err)
	}

	ln, err := net.ListenUDP("udp", listenUDPAddr)
	if err != nil {
		return err
	}
	buf := make([]byte, 4096)
	ch := make(chan string, 64)
	go c.OnlineWorker(ch)
	for {
		n, addr, err := ln.ReadFromUDP(buf)
		if err != nil {
			log.Print(err)
			continue
		}
		if !addr.IP.Equal(serverAddr) || addr.Port != c.ServerPort {
			log.Printf("Received packet from unexpected address: %s", addr)
			continue
		}
		text := strings.TrimSpace(string(buf[:n]))
		parts := strings.SplitN(text, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		ch <- parts[1]
	}
}

func (c *Client) StartLogServer(addr string) {
	go c.LogServer(addr)
}
