package common

type RconConfig struct {
	ServerAddr string `json:"server"`
	ServerPort int    `json:"port"`
	Password   string `json:"password"`
}
