package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type CsgoStatus struct {
	Map         string   `json:"map"`
	PlayerCount int      `json:"player_count"`
	BotCount    int      `json:"bot_count"`
	Players     []string `json:"players"`
}

var (
	re_players = *regexp.MustCompile(`(\d+) humans?, (\d+) bots?`)
)

func Handle206Csgo(w http.ResponseWriter, r *http.Request) {
	res, err := http.Post("http://10.255.0.9:8001/api/exec",
		"application/json",
		bytes.NewBufferString(`{"cmd": "status"}`))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	status := new(CsgoStatus)
	status.Players = make([]string, 0, 10)

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			items := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(items[0])
			value := strings.TrimSpace(items[1])
			switch key {
			case "map":
				status.Map = value
			case "players":
				matches := re_players.FindStringSubmatch(value)
				status.PlayerCount, _ = strconv.Atoi(matches[1])
				status.BotCount, _ = strconv.Atoi(matches[2])
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
