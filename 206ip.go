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
		http.Error(w, fmt.Sprintf("internal server error: %v\n", err), http.StatusInternalServerError)
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
			http.Error(w, ip, http.StatusOK)
			return
		}
	}
	http.Error(w, "server not found", http.StatusInternalServerError)
}
