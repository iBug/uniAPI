package csgo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	rcon "github.com/forewing/csgo-rcon"
	"github.com/iBug/api-ustc/common"
)

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

var GameModeMap = map[int]string{
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
	DockerHost    string `json:"docker-host"`
	ContainerName string `json:"container-name"`
	Api           string `json:"api"`
	DisableFile   string `json:"disable-file"`
}

type Client struct {
	Api        string
	CacheTime  time.Duration
	SilentFunc func() bool

	rcon   *rcon.Client
	docker *client.Client

	savedStatus  Status
	localState   LocalState
	localStateMu sync.Mutex
}

func NewClient(config Config) *Client {
	c := &Client{
		Api:       config.Api,
		CacheTime: 10 * time.Second,
		rcon:      common.RconClient(config.RconConfig),
	}

	if config.ContainerName != "" {
		c.StartLogWatcher(config.DockerHost, config.ContainerName)
	}

	if config.DisableFile != "" {
		c.SilentFunc = func() bool {
			_, err := os.Stat(config.DisableFile)
			return err == nil
		}
	}
	return c
}

func (s *Status) ParseGameMode() string {
	// Source: https://totalcsgo.com/command/gamemode
	id := s.cvar_GameType*100 + s.cvar_GameMode
	if str, ok := GameModeMap[id]; ok {
		return str
	}
	return "unknown"
}

func (c *Client) GetStatus() (Status, error) {
	msg, err := c.rcon.Execute("status; cvarlist game_")
	retries := 0
	for err != nil {
		retries++
		log.Printf("csgo.GetStatus rcon error %d: %v", retries, err)
		if retries >= 3 {
			return Status{}, fmt.Errorf("csgo.GetStatus error: %w", err)
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
		if strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "\"", 3)
			if len(parts) != 3 {
				continue
			}
			moreInfo := strings.Split(strings.TrimSpace(parts[2]), " ")
			if moreInfo[0] == "BOT" {
				continue
			}
			status.Players = append(status.Players, parts[1])
			continue
		} else if strings.HasPrefix(line, "loaded spawngroup") {
			items := strings.Split(line, "[1:")
			if len(items) < 2 {
				continue
			}
			items = strings.Split(items[1], "|")
			if len(items) == 0 {
				continue
			}
			status.Map = strings.TrimSpace(items[0])
			continue
		} else {
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
				matches := RePlayers.FindStringSubmatch(value)
				status.PlayerCount, _ = strconv.Atoi(matches[1])
				status.BotCount, _ = strconv.Atoi(matches[2])
			case "game_mode":
				status.cvar_GameMode, _ = strconv.Atoi(value)
			case "game_type":
				status.cvar_GameType, _ = strconv.Atoi(value)
			}
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

func (c *Client) GetCachedStatus() (Status, error) {
	if time.Since(c.savedStatus.Time) < c.CacheTime {
		return c.savedStatus, nil
	}
	return c.GetStatus()
}

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := c.GetStatus()
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
		retry++
		log.Printf("SendOnlineNotice error %d: %v\n", retry, err)
		if retry >= 3 {
			return err
		}
		time.Sleep(1 * time.Second)
		res, err = http.DefaultClient.Do(req)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	return nil
}

func (c *Client) handleLogMessage(s string) {
	// Check online
	matches := ReConnected.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("CSGO Online: %v connected\n", matches[1])
		status, err := c.GetStatus()
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
	matches = ReDisconnected.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("CSGO Online: %v disconnected\n", matches[1])
		c.localState.RemovePlayer(matches[1])
		status, err := c.GetStatus()
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

	// Check join team
	matches = ReJoinTeam.FindStringSubmatch(s)
	if len(matches) == 6 {
		player, oldTeam, newTeam := matches[1], matches[4], matches[5]
		c.localStateMu.Lock()
		defer c.localStateMu.Unlock()
		if matches[3] == "BOT" {
			c.localState.JoinTeam("BOT", oldTeam, newTeam)
			return
		}
		log.Printf("CSGO Online: %v joins team %v\n", player, newTeam)
		c.localState.JoinTeam(player, oldTeam, newTeam)
	}

	// Check game state
	matches = ReMatchStatus.FindStringSubmatch(s)
	if len(matches) >= 4 {
		c.localStateMu.Lock()
		c.localState.CT.Score, _ = strconv.Atoi(matches[1])
		c.localState.T.Score, _ = strconv.Atoi(matches[2])
		c.localState.Map = matches[3]
		c.localState.RoundsPlayed, _ = strconv.Atoi(matches[4])
		c.localState.GameOngoing = c.localState.RoundsPlayed >= 0
		c.localStateMu.Unlock()
		return
	}

	// Check game over
	matches = ReGameOver.FindStringSubmatch(s)
	if len(matches) > 0 {
		c.localStateMu.Lock()
		c.localState.GameOngoing = false
		c.localStateMu.Unlock()
		return
	}

	// Cleanup
	matches = ReLogFileClosed.FindStringSubmatch(s)
	if len(matches) > 0 {
		c.localStateMu.Lock()
		c.localState.UnsetTeams()
		c.localStateMu.Unlock()
		return
	}
}

func (c *Client) onlineWorker(ch <-chan string) {
	for s := range ch {
		c.handleLogMessage(s)
	}
}

func (c *Client) logWatcher(container string, ch chan<- string) error {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: false,
		Follow:     true,
		Tail:       "1",
	}
	for {
		reader, err := c.docker.ContainerLogs(context.Background(), container, options)
		if err != nil {
			return err
		}
		pipeR, pipeW := io.Pipe()
		go stdcopy.StdCopy(pipeW, io.Discard, reader)

		scanner := bufio.NewScanner(pipeR)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			parts := strings.SplitN(text, ": ", 2)
			if len(parts) != 2 {
				continue
			}
			ch <- parts[1]
		}
		log.Println(scanner.Err())
		reader.Close()
	}
}

func (c *Client) StartLogWatcher(dockerHost string, container string) error {
	d, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return err
	}
	c.docker = d
	ch := make(chan string, 1)
	go c.onlineWorker(ch)
	go func() {
		for {
			err := c.logWatcher(container, ch)
			log.Printf("CSGO log watcher error: %w\n", err)
			time.Sleep(time.Second)
		}
	}()
	return nil
}