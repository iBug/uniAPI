package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

var (
	listenAddr string
)

func main() {
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.Parse()

	// $INVOCATION_ID is set by systemd v232+
	if _, ok := os.LookupEnv("INVOCATION_ID"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	http.HandleFunc("/csgo", Handle206Csgo)
	http.HandleFunc("/206ip", Handle206IP)
	http.HandleFunc("/ibug-auth", HandleIBugAuth)

	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
