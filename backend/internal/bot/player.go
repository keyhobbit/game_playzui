package bot

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/game-playzui/tienlen-server/internal/models"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

type BotPlayer struct {
	Client        *ws.Client
	RoomID        int
	SeatIndex     int
	Difficulty    Difficulty
	hand          []models.Card
	lastTablePlay *movePlayedPayload
	stopCh        chan struct{}
}

func NewBotPlayer(client *ws.Client, roomID, seat int, diff Difficulty) *BotPlayer {
	return &BotPlayer{
		Client:     client,
		RoomID:     roomID,
		SeatIndex:  seat,
		Difficulty: diff,
		stopCh:     make(chan struct{}),
	}
}

func (bp *BotPlayer) Start() {
	go bp.listen()
}

func (bp *BotPlayer) Stop() {
	select {
	case <-bp.stopCh:
	default:
		close(bp.stopCh)
	}
}

func (bp *BotPlayer) listen() {
	for {
		select {
		case <-bp.stopCh:
			return
		case data, ok := <-bp.Client.Send:
			if !ok {
				return
			}
			bp.handleMessage(data)
		}
	}
}

func (bp *BotPlayer) handleMessage(data []byte) {
	var msg ws.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Type {
	case ws.MsgCardDealt:
		bp.onCardDealt(msg.Payload)
	case ws.MsgGameState:
		bp.onGameState(msg.Payload)
	case ws.MsgMovePlayed:
		bp.onMovePlayed(msg.Payload)
	case ws.MsgTurnChange:
		bp.onTurnChange(msg.Payload)
	case ws.MsgSettlement:
		bp.onSettlement()
	case ws.MsgRoomUpdate:
		bp.onRoomUpdate(msg.Payload)
	}
}

type cardDealtPayload struct {
	Hand        []models.Card `json:"hand"`
	CurrentTurn int           `json:"current_turn"`
	Players     []struct {
		SeatIndex int `json:"seat_index"`
	} `json:"players"`
	TablePlay *json.RawMessage `json:"table_play"`
}

func (bp *BotPlayer) onCardDealt(payload json.RawMessage) {
	var p cardDealtPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	bp.hand = p.Hand
	log.Printf("bot %s: received %d cards, current_turn=%d, my_seat=%d",
		bp.Client.Username, len(bp.hand), p.CurrentTurn, bp.SeatIndex)

	if p.CurrentTurn == bp.SeatIndex {
		bp.playTurn(nil)
	}
}

func (bp *BotPlayer) onGameState(payload json.RawMessage) {
	var p struct {
		Hand        []models.Card `json:"hand"`
		CurrentTurn int           `json:"current_turn"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if len(p.Hand) > 0 {
		bp.hand = p.Hand
	}
}

type movePlayedPayload struct {
	PlayerIndex int                  `json:"player_index"`
	Cards       []models.Card        `json:"cards"`
	ComboType   models.CombinationType `json:"combo_type"`
}

func (bp *BotPlayer) onMovePlayed(payload json.RawMessage) {
	var p movePlayedPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if p.PlayerIndex == bp.SeatIndex {
		bp.hand = models.RemoveCards(bp.hand, p.Cards)
	}
	bp.lastTablePlay = &p
}

type turnChangePayload struct {
	CurrentTurn int    `json:"current_turn"`
	TableClear  bool   `json:"table_clear"`
	Action      string `json:"action,omitempty"`
}

func (bp *BotPlayer) onTurnChange(payload json.RawMessage) {
	var p turnChangePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	if p.CurrentTurn != bp.SeatIndex {
		return
	}

	var table *TableState
	if p.TableClear {
		table = nil
	} else if bp.lastTablePlay != nil {
		table = &TableState{
			Cards:     bp.lastTablePlay.Cards,
			ComboType: bp.lastTablePlay.ComboType,
		}
	}

	bp.playTurn(table)
}

func (bp *BotPlayer) playTurn(table *TableState) {
	delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
	go func() {
		select {
		case <-time.After(delay):
		case <-bp.stopCh:
			return
		}

		play := ChoosePlay(bp.hand, table, bp.Difficulty)
		if play == nil {
			bp.sendPass()
			return
		}

		bp.sendPlay(play.Cards)
	}()
}

func (bp *BotPlayer) sendPlay(cards []models.Card) {
	cardPayloads := make([]ws.CardPayload, len(cards))
	for i, c := range cards {
		cardPayloads[i] = ws.CardPayload{Rank: c.Rank.String(), Suit: c.Suit.String()}
	}
	payload, _ := json.Marshal(ws.PlayCardsPayload{Cards: cardPayloads})
	msg := ws.Message{Type: ws.MsgPlayCards, Payload: payload}
	bp.Client.Hub.InjectMessage(bp.Client, msg)
}

func (bp *BotPlayer) sendPass() {
	payload, _ := json.Marshal(struct{}{})
	msg := ws.Message{Type: ws.MsgPassTurn, Payload: payload}
	bp.Client.Hub.InjectMessage(bp.Client, msg)
}

func (bp *BotPlayer) sendReady() {
	payload, _ := json.Marshal(struct{}{})
	msg := ws.Message{Type: ws.MsgReady, Payload: payload}
	bp.Client.Hub.InjectMessage(bp.Client, msg)
}

func (bp *BotPlayer) onSettlement() {
	bp.hand = nil
	bp.lastTablePlay = nil
	delay := time.Duration(2000+rand.Intn(1000)) * time.Millisecond
	go func() {
		select {
		case <-time.After(delay):
		case <-bp.stopCh:
			return
		}
		// Wait for room to reset to LOBBY, then auto-ready
		time.Sleep(6 * time.Second)
		bp.sendReady()
	}()
}

func (bp *BotPlayer) onRoomUpdate(payload json.RawMessage) {
	var p struct {
		Phase       string `json:"phase"`
		PlayerCount int    `json:"player_count"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	// No action needed here; settlement handler auto-readies
}
