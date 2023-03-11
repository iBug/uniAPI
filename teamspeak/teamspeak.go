package teamspeak

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
	ID           int    `json:"cid"`
	Name         string `json:"channel_name"`
	Order        int    `json:"channel_order"`
	Parent       int    `json:"pid"`
	TotalClients int    `json:"total_clients"`
}

// Only to work around stupid TeamSpeak API
// http://choly.ca/post/go-json-marshalling/
type TSChannelA TSChannel
type TSChannelS struct {
	*TSChannelA

	ID           string `json:"cid"`
	Order        string `json:"channel_order"`
	Parent       string `json:"pid"`
	TotalClients string `json:"total_clients"`
}

func (c *TSChannel) UnmarshalJSON(data []byte) error {
	aux := &TSChannelS{TSChannelA: (*TSChannelA)(c)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	c.ID, _ = strconv.Atoi(aux.ID)
	c.Order, _ = strconv.Atoi(aux.Order)
	c.Parent, _ = strconv.Atoi(aux.Parent)
	c.TotalClients, _ = strconv.Atoi(aux.TotalClients)
	return nil
}

type TSClient struct {
	ChannelID  int    `json:"cid"`
	ID         int    `json:"clid"`
	DatabaseID int    `json:"client_database_id"`
	Nickname   string `json:"client_nickname"`
	Type       int    `json:"client_type"`
}
type TSClientA TSClient
type TSClientS struct {
	*TSClientA

	ChannelID  string `json:"cid"`
	ID         string `json:"clid"`
	DatabaseID string `json:"client_database_id"`
	Type       string `json:"client_type"`
}

func (c *TSClient) UnmarshalJSON(data []byte) error {
	aux := &TSClientS{TSClientA: (*TSClientA)(c)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	c.ChannelID, _ = strconv.Atoi(aux.ChannelID)
	c.ID, _ = strconv.Atoi(aux.ID)
	c.DatabaseID, _ = strconv.Atoi(aux.DatabaseID)
	c.Type, _ = strconv.Atoi(aux.Type)
	return nil
}

type TSClientInfo struct {
	ID   string `json:"cid"`
	Away string `json:"client_away"`
}

var tshttpClient = &http.Client{
	Timeout: 500 * time.Millisecond,
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

type TSStatus struct {
	Time  time.Time `json:"time"`
	Count int       `json:"count"`

	Channels []TSChannel `json:"channels"`
	Clients  []TSClient  `json:"clients"`
}

func GetTeamspeakOnline() (result TSStatus, err error) {
	result.Time = time.Now().Truncate(time.Second)

	clients, err := TSGetClients()
	if err != nil {
		return
	}
	channels, err := TSGetChannels()
	if err != nil {
		return
	}

	// Skip "system users"
	newClients := make([]TSClient, 0, len(clients))
	for _, client := range clients {
		if client.Type == 0 {
			newClients = append(newClients, client)
		}
	}
	clients = newClients

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ID < clients[j].ID
	})
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].Parent < channels[j].Parent {
			return true
		} else if channels[i].Parent > channels[j].Parent {
			return false
		}
		return channels[i].Order < channels[j].Order
	})

	result.Channels = channels
	result.Clients = clients
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
