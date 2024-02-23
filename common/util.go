package common

import (
	"log"
	"time"
)

func ParseDurationDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Invalid timeout %s, using %s\n", s, def)
		return def
	}
	return dur
}
