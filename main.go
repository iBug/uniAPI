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
)

type Config struct {
	Teamspeak  TeamspeakConfig `json:"teamspeak"`
	UstcTokens []string        `json:"ustc-tokens"`
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

	configFile := filepath.Join(homeDir, ".config", "api-ustc.json")
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

	if csgologAddr != "" {
		StartCsgoLogServer(csgologAddr)
	}

	// Reload config on SIGHUP
	signalC := make(chan os.Signal)
	signal.Notify(signalC, syscall.SIGHUP)
	go func() {
		for range signalC {
			if err := LoadConfig(); err != nil {
				log.Printf("Error reloading config: %v", err)
			} else {
				log.Printf("Config reloaded!")
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/csgo", Handle206Csgo)
	mux.HandleFunc("/minecraft", Handle206Minecraft)
	mux.HandleFunc("/factorio", Handle206Factorio)
	mux.HandleFunc("/teamspeak", HandleTeamspeakOnline)
	mux.HandleFunc("/206ip", Handle206IP)
	mux.HandleFunc("/ibug-auth", HandleIBugAuth)
	mux.HandleFunc("/ustc-id", HandleUstcId)
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "User-Agent: *\nDisallow: /\n")
	})

	outerMux := http.NewServeMux()
	outerMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		w.Header().Set("X-Robots-Tag", "noindex")
		mux.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(listenAddr, outerMux))
}
