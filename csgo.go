package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CsgoStatus struct {
	Time        time.Time `json:"time"`
	Map         string    `json:"map"`
	GameMode    string    `json:"game_mode"`
	PlayerCount int       `json:"player_count"`
	BotCount    int       `json:"bot_count"`
	Players     []string  `json:"players"`

	cvar_GameMode int
	cvar_GameType int
}

type CsgoOnlinePayload struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

var RE_PLAYERS = *regexp.MustCompile(`(\d+) humans?, (\d+) bots?`)
var RE_CONNECTED = *regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>" connected,`)
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
}

var (
	SavedCsgoStatus CsgoStatus
)

const (
	CSGO_RCON_API    = "http://10.255.0.9:8001/api/exec/"
	CSGO_SERVER_ADDR = "10.255.0.9"
	CSGO_SERVER_PORT = 27015
	CSGO_ONLINE_API  = "https://api.ibugone.com/gh/206steam"
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
	if useCache && time.Now().Sub(SavedCsgoStatus.Time) < 10*time.Second {
		return SavedCsgoStatus, nil
	}

	res, err := http.Post(CSGO_RCON_API,
		"application/json",
		bytes.NewBufferString(`{"cmd": "cvarlist game_; status"}`))
	if err != nil {
		return CsgoStatus{}, err
	}
	defer res.Body.Close()

	status := CsgoStatus{Players: make([]string, 0, 10)}
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

	status.Time = time.Now().Truncate(time.Second)
	SavedCsgoStatus = status
	return status, nil
}

func Handle206Csgo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := GetCsgoStatus(true)
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

func CsgoSendOnlineNotice(name string, count int) error {
	payloadObj := CsgoOnlinePayload{Action: "goonline", Name: name, Count: count}
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
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func CsgoLogServer(listenAddr string) error {
	serverAddr := net.ParseIP(CSGO_SERVER_ADDR)
	listenUDPAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	ln, err := net.ListenUDP("udp", listenUDPAddr)
	if err != nil {
		return err
	}
	buf := make([]byte, 4096)
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
		if strings.Contains(parts[1], " connected,") || strings.Contains(parts[1], " entered the game ") {
			// log.Print(parts[1])
		}
		matches := RE_CONNECTED.FindStringSubmatch(parts[1])
		if len(matches) >= 5 && matches[3] != "BOT" {
			log.Printf("%v connected\n", matches[1])
			status, err := GetCsgoStatus(false)
			if err != nil {
				log.Print(err)
				continue
			}
			if status.PlayerCount != 1 && status.PlayerCount != 2 {
				continue
			}
			err = CsgoSendOnlineNotice(matches[1], status.PlayerCount)
			if err != nil {
				log.Print(err)
				continue
			}
		}
	}
}

func StartCsgoLogServer(addr string) {
	go CsgoLogServer(addr)
}
