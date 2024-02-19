package teamspeak

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/iBug/api-ustc/common"
)

type Config struct {
	Key      string `json:"key"`
	Instance string `json:"instance"`
	Endpoint string `json:"endpoint"`
	Timeout  string `json:"timeout"`
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
type tsChannelA TSChannel
type tsChannelS struct {
	*tsChannelA

	ID           string `json:"cid"`
	Order        string `json:"channel_order"`
	Parent       string `json:"pid"`
	TotalClients string `json:"total_clients"`
}

func (c *TSChannel) UnmarshalJSON(data []byte) error {
	aux := &tsChannelS{tsChannelA: (*tsChannelA)(c)}
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
type tsClientA TSClient
type tsClientS struct {
	*tsClientA

	ChannelID  string `json:"cid"`
	ID         string `json:"clid"`
	DatabaseID string `json:"client_database_id"`
	Type       string `json:"client_type"`
}

func (c *TSClient) UnmarshalJSON(data []byte) error {
	aux := &tsClientS{tsClientA: (*tsClientA)(c)}
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

type Client struct {
	endpoint   string
	instance   string
	key        string
	httpClient *http.Client
}

func NewClient(config Config) *Client {
	return &Client{
		endpoint: config.Endpoint,
		instance: config.Instance,
		key:      config.Key,
		httpClient: &http.Client{
			Timeout: common.ParseDurationDefault(config.Timeout, 1*time.Second),
		},
	}
}

func (c *Client) QueryHTTP(method string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s/%s/%s", c.endpoint, c.instance, method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "iBug.api-ustc/dev")
	req.Header.Set("X-API-Key", c.key)
	return c.httpClient.Do(req)
}

func (c *Client) Query(method string, body any) error {
	retries := 0
	resp, err := c.QueryHTTP(method)
	for err != nil {
		retries++
		log.Printf("Teamspeak query error %d: %v", retries, err)
		if retries >= 3 {
			return err
		}
		resp, err = c.QueryHTTP(method)
	}
	defer resp.Body.Close()

	result := TSQueryResponse{Body: body}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetClients() ([]TSClient, error) {
	clients := make([]TSClient, 0)
	err := c.Query("clientlist", &clients)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func (c *Client) GetChannels() ([]TSChannel, error) {
	channels := make([]TSChannel, 0)
	err := c.Query("channellist", &channels)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

type Status struct {
	Time  time.Time `json:"time"`
	Count int       `json:"count"`

	Channels []TSChannel `json:"channels"`
	Clients  []TSClient  `json:"clients"`
}

func (c *Client) GetOnline() (result Status, err error) {
	result.Time = time.Now().Truncate(time.Second)

	clients, err := c.GetClients()
	if err != nil {
		return
	}
	channels, err := c.GetChannels()
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

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := c.GetOnline()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func NewService(config json.RawMessage) (common.Service, error) {
	var cfg Config
	err := json.Unmarshal(config, &cfg)
	if err != nil {
		return nil, err
	}
	return NewClient(cfg), nil
}

func init() {
	common.RegisterService("teamspeak", NewService)
}
