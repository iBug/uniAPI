package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

const WG_PUBKEY_206 = "9oG+6Ba3x9KOw5D6xfOXzE0qOJFc+WhNlGV9PUGpSDc="

func Handle206IP(w http.ResponseWriter, req *http.Request) {
	outb, err := exec.Command("/usr/bin/sudo", "/usr/bin/wg", "show", "wg0", "endpoints").Output()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "internal server error: %s\n", err)
		return
	}
	out := strings.Split(string(outb), "\n")
	for _, line := range out {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			break
		}
		if parts[0] == WG_PUBKEY_206 {
			ip := strings.Split(parts[1], ":")[0]
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(ip + "\n"))
			return
		}
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("server not found\n"))
}
