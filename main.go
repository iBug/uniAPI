package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/iBug/api-ustc/common"
	"github.com/iBug/api-ustc/csgo"
	"github.com/iBug/api-ustc/factorio"
	"github.com/iBug/api-ustc/ibugauth"
	"github.com/iBug/api-ustc/minecraft"
	"github.com/iBug/api-ustc/teamspeak"
	"github.com/iBug/api-ustc/terraria"
	"github.com/iBug/api-ustc/ustc"
)

type RconConfig struct {
	ServerAddr string `json:"server"`
	ServerPort int    `json:"port"`
	Password   string `json:"password"`
}

type CsgoConfig struct {
	RconConfig
	Api         string `json:"api"`
	DisableFile string `json:"disable-file"`
}

type TerrariaConfig struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type Config struct {
	Csgo       CsgoConfig                `json:"csgo"`
	Factorio   RconConfig                `json:"factorio"`
	Minecraft  RconConfig                `json:"minecraft"`
	Terraria   TerrariaConfig            `json:"terraria"`
	Teamspeak  teamspeak.TeamspeakConfig `json:"teamspeak"`
	UstcTokens []string                  `json:"ustc-tokens"`
	WgPubkey   string                    `json:"wg-pubkey"`
}

var (
	listenAddr  string
	csgologAddr string

	config Config
)

func LogRequest(r *http.Request) {
	remoteAddr := r.Header.Get("CF-Connecting-IP")
	if remoteAddr == "" {
		remoteAddr = "(local)"
	}
	log.Printf("%s %q from %s\n", r.Method, r.URL.Path, remoteAddr)
}

func LoadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configFile := filepath.Join(homeDir, ".config/api-ustc.json")
	f, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.StringVar(&csgologAddr, "csgolog", "", "CS:GO log listen address")
	flag.Parse()

	// $JOURNAL_STREAM is set by systemd v231+
	if _, ok := os.LookupEnv("JOURNAL_STREAM"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if err := LoadConfig(); err != nil {
		log.Fatal(err)
	}

	csgoClient := csgo.NewClient(config.Csgo.ServerAddr, config.Csgo.ServerPort, config.Csgo.Password, 100*time.Millisecond)
	csgoClient.Api = config.Csgo.Api
	csgoClient.SilentFunc = func() bool {
		_, err := os.Stat(config.Csgo.DisableFile)
		return err == nil
	}
	if csgologAddr != "" {
		csgoClient.StartLogServer(csgologAddr)
	}

	facClient := factorio.NewClient(config.Factorio.ServerAddr, config.Factorio.ServerPort, config.Factorio.Password, 100*time.Millisecond)

	minecraftClient := minecraft.NewClient(config.Minecraft.ServerAddr, config.Minecraft.ServerPort, config.Minecraft.Password, 10*time.Millisecond)

	trClient := terraria.NewClient(config.Terraria.Host, config.Terraria.Container)

	tsClient := teamspeak.NewClient(config.Teamspeak.Endpoint, config.Teamspeak.Instance, config.Teamspeak.Key, 500*time.Millisecond)
	tsHandler := &common.TokenProtectedHandler{tsClient, config.UstcTokens}

	ustcHandler := &common.TokenProtectedHandler{http.HandlerFunc(ustc.HandleUstcId), config.UstcTokens}

	// Reload config on SIGHUP
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGHUP)
	go func() {
		for range signalC {
			if err := LoadConfig(); err != nil {
				log.Printf("Error reloading config: %v", err)
			} else {
				tsHandler.Tokens = config.UstcTokens
				ustcHandler.Tokens = config.UstcTokens
				log.Printf("Config reloaded!")
			}
		}
	}()

	mainMux := http.NewServeMux()
	mainMux.Handle("/csgo", csgoClient)
	mainMux.Handle("/factorio", facClient)
	mainMux.Handle("/minecraft", minecraftClient)
	mainMux.Handle("/terraria", trClient)
	mainMux.Handle("/teamspeak", tsHandler)
	mainMux.HandleFunc("/206ip", Handle206IP)
	mainMux.HandleFunc("/ibug-auth", ibugauth.HandleIBugAuth)
	mainMux.Handle("/ustc-id", ustcHandler)
	mainMux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "User-Agent: *\nDisallow: /\n")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		w.Header().Set("X-Robots-Tag", "noindex")
		mainMux.ServeHTTP(w, r)
	})
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}
