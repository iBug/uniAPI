package main

import "flag"

var (
	listenAddr string
)

func main() {
	flag.StringVar(&listenAddr, "l", ":8000", "listen address")
	flag.Parse()
}
