package csgo

import "regexp"

var RePlayers = *regexp.MustCompile(`(\d+) humans?, (\d+) bots?`)
var ReConnected = *regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>" connected,`)
var ReDisconnected = *regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>" disconnected \(`)
var ReMatchStatus = *regexp.MustCompile(`MatchStatus: Score: (\d+):(\d+) on map "(\w+)" RoundsPlayed: (\d+)`)
var ReGameOver = *regexp.MustCompile(`^(Game Over:)`)
