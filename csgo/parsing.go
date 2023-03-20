package csgo

import (
	"fmt"
	"regexp"
)

const (
	rePlayer3 = `"([^<]+)<(\d+)><([^>]+)>"`
	rePlayer4 = `"([^<]+)<(\d+)><([^>]+)><([^>]*)>"`
)

const (
	teamUnassigned = "Unassigned"
	teamCT         = "CT"
	teamT          = "TERRORIST"
	teamSpectator  = "SPECTATOR"
)

func r(format string, args ...any) regexp.Regexp {
	return *regexp.MustCompile(fmt.Sprintf(format, args...))
}

var (
	RePlayers      = r(`^(\d+) humans?, (\d+) bots?`)
	ReConnected    = r(`^%s connected,`, rePlayer4)
	ReDisconnected = r(`^%s disconnected \(`, rePlayer4)
	ReJoinTeam     = r(`^%s switched from team <(\w+)> to <(\w+)>`, rePlayer3)
	ReMatchStatus  = r(`^MatchStatus: Score: (\d+):(\d+) on map "(\w+)" RoundsPlayed: (\d+)`)
	ReGameOver     = r(`^(Game Over:)`)

	ReLogFileClosed = r(`^Log file closed`)
)
