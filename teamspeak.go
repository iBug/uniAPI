package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type TeamspeakConfig struct {
	Key      string `json:"key"`
	Instance string `json:"instance"`
	Endpoint string `json:"endpoint"`
}

type TSQueryResponse struct {
	Status struct {
		Code         int    `json:"code"`
		Message      string `json:"message"`
		ExtraMessage string `json:"extra_message"`
	} `json:"status"`
	Body any `json:"body"`
}

type TSChannel struct {
	ID           string `json:"cid"`
	Name         string `json:"channel_name"`
	Order        string `json:"channel_order"`
	Parent       string `json:"pid"`
	TotalClients string `json:"total_clients"`
}

type TSClient struct {
	ChannelID  string `json:"cid"`
	ID         string `json:"clid"`
	DatabaseID string `json:"client_database_id"`
	Nickname   string `json:"client_nickname"`
	Type       string `json:"client_type"`
}

var tshttpClient = &http.Client{
	Timeout: 150 * time.Millisecond,
}

func TSQueryHTTP(method string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s/%s/%s", config.Teamspeak.Endpoint, config.Teamspeak.Instance, method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "iBug.api-ustc/dev")
	req.Header.Set("X-API-Key", config.Teamspeak.Key)
	return tshttpClient.Do(req)
}

func TSQuery(method string, body any) error {
	resp, err := TSQueryHTTP(method)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result := TSQueryResponse{Body: body}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	return nil
}

func TSGetClients() ([]TSClient, error) {
	clients := make([]TSClient, 0)
	err := TSQuery("clientlist", &clients)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func TSGetChannels() ([]TSChannel, error) {
	channels := make([]TSChannel, 0)
	err := TSQuery("channellist", &channels)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

type TSOnlineChannel struct {
	Name    string   `json:"name"`
	Order   int      `json:"-"`
	Clients []string `json:"clients"`
}

type TSOnlinePayload struct {
	Count int `json:"count"`

	Channels []TSOnlineChannel `json:"channels"`
}

func GetTeamspeakOnline() (result TSOnlinePayload, err error) {
	clients, err := TSGetClients()
	if err != nil {
		return
	}
	channels, err := TSGetChannels()
	if err != nil {
		return
	}

	// Build a map of channel ID to channel name
	channelMap := make(map[string]*TSOnlineChannel)
	for _, channel := range channels {
		order, err := strconv.Atoi(channel.Order)
		if err != nil {
			order = 0
		}
		channelMap[channel.ID] = &TSOnlineChannel{
			Name:  channel.Name,
			Order: order,
		}
	}

	// Build a map of channel ID to clients
	for _, client := range clients {
		if client.Type != "0" {
			continue
		}
		channelMap[client.ChannelID].Clients = append(channelMap[client.ChannelID].Clients, client.Nickname)
	}

	// Build channel list
	channelList := make([]TSOnlineChannel, 0)
	for _, channel := range channelMap {
		if len(channel.Clients) == 0 {
			continue
		}
		result.Count += len(channel.Clients)
		channelList = append(channelList, *channel)
	}

	// Sort channels
	sort.Slice(channelList, func(i, j int) bool {
		return channelList[i].Order < channelList[j].Order
	})

	result.Channels = channelList
	return
}

func HandleTeamspeakOnline(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("CF-Connecting-IP") != "" &&
		!ValidateToken(r.Header.Get("Authorization"), config.UstcTokens) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	result, err := GetTeamspeakOnline()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
