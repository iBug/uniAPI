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
)

type MinecraftStatus struct {
	Time     time.Time `json:"time"`
	Count    int       `json:"count"`
	MaxCount int       `json:"max_count"`
	Players  []string  `json:"players"`
}

const (
	MC_SERVER_ADDR = "192.168.3.2"
	MC_RCON_PORT   = 25575
	MC_RCON_PASS   = "taokystrong"
)

var RE_MC_LIST = *regexp.MustCompile(`^There are (\d+) of a max of (\d+) players online: `)

var (
	mcRcon = rcon.New(fmt.Sprintf("%s:%d", MC_SERVER_ADDR, MC_RCON_PORT), MC_RCON_PASS, time.Millisecond*10)
)

func GetMinecraftStatus() (MinecraftStatus, error) {
	status := MinecraftStatus{}
	msg, err := mcRcon.Execute("list")
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

func Handle206Minecraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := GetMinecraftStatus()
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
