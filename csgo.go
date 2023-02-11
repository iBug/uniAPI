package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rcon "github.com/forewing/csgo-rcon"
)

type CsgoLocalState struct {
	CTScore      int `json:"ct_score"`
	TScore       int `json:"t_score"`
	Map          string
	RoundsPlayed int  `json:"rounds_played"`
	GameOngoing  bool `json:"game_ongoing"`
}

type CsgoStatus struct {
	Time        time.Time `json:"time"`
	Map         string    `json:"map"`
	GameMode    string    `json:"game_mode"`
	PlayerCount int       `json:"player_count"`
	BotCount    int       `json:"bot_count"`
	Players     []string  `json:"players"`

	LocalState CsgoLocalState `json:"local_state"`

	cvar_GameMode int
	cvar_GameType int
}

type CsgoOnlinePayload struct {
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
	100: "arms race",
	101: "demolition",
	102: "deathmatch",
	200: "training",
	300: "custom",
	400: "cooperative",
	500: "skirmish",
	600: "danger zone",
}

var (
	savedCsgoStatus CsgoStatus
	localState      CsgoLocalState
	localStateMu    sync.Mutex

	csgoRcon = rcon.New(fmt.Sprintf("%s:%d", CSGO_SERVER_ADDR, CSGO_SERVER_PORT),
		CSGO_RCON_PASS,
		time.Millisecond*100)
)

const (
	CSGO_RCON_API     = "http://10.255.0.9:8001/api/exec/"
	CSGO_RCON_PASS    = "pointeeserver"
	CSGO_SERVER_ADDR  = "10.255.0.9"
	CSGO_SERVER_PORT  = 27015
	CSGO_ONLINE_API   = "https://api.ibugone.com/gh/206steam"
	CSGO_DISABLE_FILE = "/tmp/noonline"

	CSGO_CACHE_TIME = 10 * time.Second
)

func (s CsgoStatus) ParseGameMode() string {
	// Source: https://totalcsgo.com/command/gamemode
	id := s.cvar_GameType*100 + s.cvar_GameMode
	str, ok := GAME_MODE_S[id]
	if ok {
		return str
	}
	return "unknown"
}

func GetCsgoStatus(useCache bool) (CsgoStatus, error) {
	if useCache && time.Since(savedCsgoStatus.Time) < CSGO_CACHE_TIME {
		return savedCsgoStatus, nil
	}

	msg, err := csgoRcon.Execute("cvarlist game_; status")
	if err != nil {
		return CsgoStatus{}, err
	}

	status := CsgoStatus{Players: make([]string, 0, 10)}
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

	localStateMu.Lock()
	status.LocalState = localState
	localStateMu.Unlock()

	status.Time = time.Now().Truncate(time.Second)
	savedCsgoStatus = status
	return status, nil
}

func Handle206Csgo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := GetCsgoStatus(true)
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

func CsgoShouldSuppressNotice() bool {
	_, err := os.Stat(CSGO_DISABLE_FILE)
	return err == nil
}

func CsgoSendOnlineNotice(action, name string, count int) error {
	if CsgoShouldSuppressNotice() {
		return nil
	}
	payloadObj := CsgoOnlinePayload{Action: action, Name: name, Count: count}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", CSGO_ONLINE_API, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "ping")
	res, err := http.DefaultClient.Do(req)
	retry := 0
	for err != nil {
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

func CsgoHandleLogMessage(s string) {
	// Check online
	matches := RE_CONNECTED.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("%v connected\n", matches[1])
		status, err := GetCsgoStatus(false)
		if err != nil {
			log.Print(err)
			return
		}
		if status.PlayerCount < 1 || status.PlayerCount > 2 {
			return
		}
		err = CsgoSendOnlineNotice("goonline", matches[1], status.PlayerCount)
		if err != nil {
			log.Print(err)
		}
		return
	}

	// Check offline
	matches = RE_DISCONNECTED.FindStringSubmatch(s)
	if len(matches) >= 5 && matches[3] != "BOT" {
		log.Printf("%v disconnected\n", matches[1])
		status, err := GetCsgoStatus(false)
		if err != nil {
			log.Print(err)
			return
		}
		if status.PlayerCount > 0 {
			return
		}
		err = CsgoSendOnlineNotice("gooffline", matches[1], status.PlayerCount)
		if err != nil {
			log.Print(err)
		}
		return
	}

	// Check game state
	matches = RE_MATCH_STATUS.FindStringSubmatch(s)
	if len(matches) >= 4 {
		localStateMu.Lock()
		localState.CTScore, _ = strconv.Atoi(matches[1])
		localState.TScore, _ = strconv.Atoi(matches[2])
		localState.Map = matches[3]
		localState.RoundsPlayed, _ = strconv.Atoi(matches[4])
		localState.GameOngoing = localState.RoundsPlayed >= 0
		localStateMu.Unlock()
		return
	}

	// Check game over
	matches = RE_GAME_OVER.FindStringSubmatch(s)
	if len(matches) > 0 {
		localStateMu.Lock()
		localState.GameOngoing = false
		localStateMu.Unlock()
		return
	}
}

func CsgoOnlineWorker(ch <-chan string) {
	for s := range ch {
		CsgoHandleLogMessage(s)
	}
}

func CsgoLogServer(listenAddr string) error {
	serverAddr := net.ParseIP(CSGO_SERVER_ADDR)
	listenUDPAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return err
	}

	ln, err := net.ListenUDP("udp", listenUDPAddr)
	if err != nil {
		return err
	}
	buf := make([]byte, 4096)
	ch := make(chan string, 64)
	go CsgoOnlineWorker(ch)
	for {
		n, addr, err := ln.ReadFromUDP(buf)
		if err != nil {
			log.Print(err)
			continue
		}
		if !addr.IP.Equal(serverAddr) || addr.Port != CSGO_SERVER_PORT {
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

func StartCsgoLogServer(addr string) {
	go CsgoLogServer(addr)
}
