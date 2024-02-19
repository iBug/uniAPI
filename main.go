package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iBug/api-ustc/common"
	_ "github.com/iBug/api-ustc/plugins"
	"github.com/iBug/api-ustc/plugins/ibugauth"
)

type Config struct {
	Services ServiceSet `json:"services"`
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

func LoadConfig(path string) error {
	if path == "" {
		var err error
		path, err = common.DefaultConfigPath()
		if err != nil {
			return err
		}

	}
	f, err := os.Open(path)
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
	var configFile string
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.StringVar(&configFile, "c", "", "config file (default ~/.config/api-ustc.yml)")
	flag.Parse()

	// $JOURNAL_STREAM is set by systemd v231+
	if _, ok := os.LookupEnv("JOURNAL_STREAM"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if err := LoadConfig(configFile); err != nil {
		log.Fatal(err)
	}

	// Reload config on SIGHUP
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGHUP)
	go func() {
		for range signalC {
			if err := LoadConfig(configFile); err != nil {
				log.Printf("Error reloading config: %v", err)
			} else {
				log.Printf("Config reloaded!")
			}
		}
	}()

	mainMux := http.NewServeMux()
	mainMux.Handle("/csgo", csgoClient)
	mainMux.Handle("/factorio", facClient)
	mainMux.Handle("/minecraft", minecraftClient)
	mainMux.Handle("/palworld", palworldClient)
	mainMux.Handle("/terraria", trClient)
	mainMux.Handle("/teamspeak", tsHandler)
	mainMux.HandleFunc("/206ip", Handle206IP)
	mainMux.HandleFunc("/ibug-auth", ibugauth.HandleIBugAuth)
	mainMux.Handle("/ustc-id", ustcHandler)
	mainMux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {})

	handler := func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		w.Header().Set("X-Robots-Tag", "noindex")
		mainMux.ServeHTTP(w, r)
	}
	s := &http.Server{
		Addr:         listenAddr,
		Handler:      http.HandlerFunc(handler),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(s.ListenAndServe())
}
