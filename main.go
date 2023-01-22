package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

var (
	listenAddr  string
	csgologAddr string
)

func LogRequest(r *http.Request) {
	remoteAddr := r.Header.Get("CF-Connecting-IP")
	log.Printf("%s %q from %s\n", r.Method, r.URL.Path, remoteAddr)
}

func main() {
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.StringVar(&csgologAddr, "csgolog", "", "CS:GO log listen address")
	flag.Parse()

	// $JOURNAL_STREAM is set by systemd v231+
	if _, ok := os.LookupEnv("JOURNAL_STREAM"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if csgologAddr != "" {
		StartCsgoLogServer(csgologAddr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/csgo", Handle206Csgo)
	mux.HandleFunc("/minecraft", Handle206Minecraft)
	mux.HandleFunc("/factorio", Handle206Factorio)
	mux.HandleFunc("/206ip", Handle206IP)
	mux.HandleFunc("/ibug-auth", HandleIBugAuth)

	outerMux := http.NewServeMux()
	outerMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		mux.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(listenAddr, outerMux))
}
