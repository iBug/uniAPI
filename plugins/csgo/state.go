package csgo

type TeamStatus struct {
	Bots    int      `json:"bots"`
	Players []string `json:"players"`
	Score   int      `json:"score"`
}

type LocalState struct {
	CT           TeamStatus `json:"ct"`
	T            TeamStatus `json:"t"`
	Map          string     `json:"-"`
	RoundsPlayed int        `json:"rounds_played"`
	GameOngoing  bool       `json:"game_ongoing"`
}

func (s *LocalState) JoinTeam(player, oldTeam, newTeam string) {
	if player == "BOT" {
		switch oldTeam {
		case teamCT:
			s.CT.Bots--
		case teamT:
			s.T.Bots--
		}
		switch newTeam {
		case teamCT:
			s.CT.Bots++
		case teamT:
			s.T.Bots++
		}
		return
	}

	var players *[]string
	switch oldTeam {
	case teamCT:
		players = &s.CT.Players
	case teamT:
		players = &s.T.Players
	}
	if players != nil {
		for i, name := range *players {
			if name == player {
				*players = append((*players)[:i], (*players)[i+1:]...)
				break
			}
		}
	}

	players = nil
	switch newTeam {
	case teamCT:
		players = &s.CT.Players
	case teamT:
		players = &s.T.Players
	}
	if players != nil {
		*players = append(*players, player)
	}
}

func (s *LocalState) RemovePlayer(player string) {
	s.JoinTeam(player, teamCT, teamUnassigned)
	s.JoinTeam(player, teamT, teamUnassigned)
}

func (s *LocalState) UnsetTeams() {
	s.CT.Bots = 0
	s.CT.Players = nil
	s.T.Bots = 0
	s.T.Players = nil
}
