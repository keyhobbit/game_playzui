package models

import (
	"sync"
	"time"
)

type GamePhase string

const (
	PhaseLobby      GamePhase = "LOBBY"
	PhaseReady      GamePhase = "READY"
	PhaseDealing    GamePhase = "DEALING"
	PhasePlaying    GamePhase = "PLAYING"
	PhaseSettlement GamePhase = "SETTLEMENT"
)

type CombinationType int

const (
	ComboSingle         CombinationType = iota
	ComboPair
	ComboTriple
	ComboSequence
	ComboDoubleSequence
	ComboFourOfAKind
)

type Player struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Hand      []Card `json:"-"`
	CardCount int    `json:"card_count"`
	SeatIndex int    `json:"seat_index"`
	IsReady   bool   `json:"is_ready"`
	IsBot     bool   `json:"is_bot"`
}

type Spectator struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

type TablePlay struct {
	PlayerIndex int             `json:"player_index"`
	Cards       []Card          `json:"cards"`
	ComboType   CombinationType `json:"combo_type"`
}

type Room struct {
	mu           sync.RWMutex
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	AnteAmount   int          `json:"ante_amount"`
	Phase        GamePhase    `json:"phase"`
	Players      [4]*Player   `json:"players"`
	Spectators   []*Spectator `json:"spectators"`
	CurrentTurn  int          `json:"current_turn"`
	TablePlay    *TablePlay   `json:"table_play"`
	PassCount    int          `json:"pass_count"`
	Winner       int          `json:"winner"`
	TurnTimer    int          `json:"turn_timer"`
	HasBots      bool         `json:"has_bots"`
	WaitingSince *time.Time   `json:"-"`
}

const MaxSpectators = 3

func NewRoom(id int, name string, ante int) *Room {
	return &Room{
		ID:         id,
		Name:       name,
		AnteAmount: ante,
		Phase:      PhaseLobby,
		Spectators: make([]*Spectator, 0, MaxSpectators),
		Winner:     -1,
	}
}

func (r *Room) Lock()    { r.mu.Lock() }
func (r *Room) Unlock()  { r.mu.Unlock() }
func (r *Room) RLock()   { r.mu.RLock() }
func (r *Room) RUnlock() { r.mu.RUnlock() }

func (r *Room) PlayerCount() int {
	count := 0
	for _, p := range r.Players {
		if p != nil {
			count++
		}
	}
	return count
}

func (r *Room) FindEmptySeat() int {
	for i, p := range r.Players {
		if p == nil {
			return i
		}
	}
	return -1
}

func (r *Room) FindPlayerByUserID(userID int64) (int, *Player) {
	for i, p := range r.Players {
		if p != nil && p.UserID == userID {
			return i, p
		}
	}
	return -1, nil
}

func (r *Room) AllPlayersReady() bool {
	if r.PlayerCount() != 4 {
		return false
	}
	for _, p := range r.Players {
		if p == nil || !p.IsReady {
			return false
		}
	}
	return true
}

func (r *Room) AddSpectator(s *Spectator) bool {
	if len(r.Spectators) >= MaxSpectators {
		return false
	}
	r.Spectators = append(r.Spectators, s)
	return true
}

func (r *Room) RemoveSpectator(userID int64) {
	for i, s := range r.Spectators {
		if s.UserID == userID {
			r.Spectators = append(r.Spectators[:i], r.Spectators[i+1:]...)
			return
		}
	}
}

// RoomInfo is the public view of a room for the lobby list
type RoomInfo struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	AnteAmount  int       `json:"ante_amount"`
	Phase       GamePhase `json:"phase"`
	PlayerCount int       `json:"player_count"`
	Spectators  int       `json:"spectator_count"`
	HasBots     bool      `json:"has_bots"`
}

func (r *Room) ToInfo() RoomInfo {
	return RoomInfo{
		ID:          r.ID,
		Name:        r.Name,
		AnteAmount:  r.AnteAmount,
		Phase:       r.Phase,
		PlayerCount: r.PlayerCount(),
		Spectators:  len(r.Spectators),
		HasBots:     r.HasBots,
	}
}

func (r *Room) HumanPlayerCount() int {
	count := 0
	for _, p := range r.Players {
		if p != nil && !p.IsBot {
			count++
		}
	}
	return count
}
