package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func Handle206IP(w http.ResponseWriter, req *http.Request) {
	cmd := exec.Command("/usr/bin/sudo", "/usr/bin/wg", "show", "wg0", "endpoints")
	r, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "internal server error: %s\n", err)
		return
	}

	cmd.Start()
	defer cmd.Wait()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			break
		}
		if parts[0] == config.WgPubkey {
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
