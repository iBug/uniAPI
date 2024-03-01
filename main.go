package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iBug/uniAPI/common"
	_ "github.com/iBug/uniAPI/plugins"
	"github.com/iBug/uniAPI/server"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Services server.ServiceSet `json:"services"`
}

var handler server.ReloadableHandler

func logRequest(r *http.Request) {
	remoteAddr := r.Header.Get("CF-Connecting-IP")
	if remoteAddr == "" {
		remoteAddr = "(local)"
	}
	log.Printf("%s %q from %s\n", r.Method, r.URL.Path, remoteAddr)
}

func loadConfig(path string) error {
	if path == "" {
		var err error
		path, err = common.DefaultConfigPath()
		if err != nil {
			return err
		}

	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return err
	}

	s, err := server.NewServer(config.Services)
	if err != nil {
		return err
	}
	handler.Set(s)
	return nil
}

func main() {
	var (
		listenAddr string
		configFile string
	)
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.StringVar(&configFile, "c", "", "config file (default ~/.config/uniAPI.yml)")
	flag.Parse()

	// $JOURNAL_STREAM is set by systemd v231+
	if _, ok := os.LookupEnv("JOURNAL_STREAM"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if err := loadConfig(configFile); err != nil {
		log.Fatal(err)
	}

	// Reload config on SIGHUP
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGHUP)
	go func() {
		for range signalC {
			if err := loadConfig(configFile); err != nil {
				log.Printf("Error reloading config: %v", err)
			} else {
				log.Printf("Config reloaded!")
			}
		}
	}()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		w.Header().Set("X-Robots-Tag", "noindex")
		handler.ServeHTTP(w, r)
	})
	s := &http.Server{
		Addr:         listenAddr,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(s.ListenAndServe())
}
