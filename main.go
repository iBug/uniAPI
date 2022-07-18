package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	listenAddr string

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

	response := make(map[string]string)
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		items := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(items[0])
		value := strings.TrimSpace(items[1])
		switch key {
		case "map":
			response["map"] = value
		case "players":
			matches := re_players.FindStringSubmatch(value)
			response["players"] = matches[1]
			response["bots"] = matches[2]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.Parse()

	// $INVOCATION_ID is set by systemd v232+
	if _, ok := os.LookupEnv("INVOCATION_ID"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	http.HandleFunc("/csgo", Handle206Csgo)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
