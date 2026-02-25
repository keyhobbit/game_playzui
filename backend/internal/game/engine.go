package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/game-playzui/tienlen-server/internal/models"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

const TurnTimeout = 30 * time.Second

type MatchRequester interface {
	RequestMatch(client *ws.Client, anteLevel int)
}

type Engine struct {
	hub        *ws.Hub
	mm         MatchRequester
	turnTimers map[int]*time.Timer
}

func NewEngine(hub *ws.Hub, mm MatchRequester) *Engine {
	e := &Engine{
		hub:        hub,
		mm:         mm,
		turnTimers: make(map[int]*time.Timer),
	}
	hub.OnMessage = e.HandleMessage
	return e
}

func (e *Engine) HandleMessage(client *ws.Client, msg ws.Message) {
	switch msg.Type {
	case ws.MsgJoinRoom:
		e.handleJoinRoom(client, msg.Payload)
	case ws.MsgLeaveRoom:
		e.handleLeaveRoom(client)
	case ws.MsgReady:
		e.handleReady(client)
	case ws.MsgPlayCards:
		e.handlePlayCards(client, msg.Payload)
	case ws.MsgPassTurn:
		e.handlePassTurn(client)
	case ws.MsgChat:
		e.handleChat(client, msg.Payload)
	case ws.MsgAutoMatch:
		e.handleAutoMatch(client, msg.Payload)
	}
}

func (e *Engine) handleJoinRoom(client *ws.Client, payload json.RawMessage) {
	var p ws.JoinRoomPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.Send <- ws.NewErrorMessage("invalid join_room payload")
		return
	}

	if currentRoom := client.GetRoom(); currentRoom > 0 {
		client.Send <- ws.NewErrorMessage("already in a room, leave first")
		return
	}

	room := e.hub.GetRoom(p.RoomID)
	if room == nil {
		client.Send <- ws.NewErrorMessage("room not found")
		return
	}

	room.Lock()

	if room.Phase != models.PhaseLobby {
		if room.AddSpectator(&models.Spectator{UserID: client.UserID, Username: client.Username}) {
			client.SetRoom(p.RoomID)
			data, _ := ws.NewMessage(ws.MsgRoomUpdate, e.buildRoomState(room, -1))
			room.Unlock()
			client.Send <- data
		} else {
			room.Unlock()
			client.Send <- ws.NewErrorMessage("room is full (game in progress)")
		}
		return
	}

	seat := room.FindEmptySeat()
	if seat < 0 {
		if room.AddSpectator(&models.Spectator{UserID: client.UserID, Username: client.Username}) {
			client.SetRoom(p.RoomID)
			data, _ := ws.NewMessage(ws.MsgRoomUpdate, e.buildRoomState(room, -1))
			room.Unlock()
			client.Send <- data
		} else {
			room.Unlock()
			client.Send <- ws.NewErrorMessage("room is full")
		}
		return
	}

	room.Players[seat] = &models.Player{
		UserID:    client.UserID,
		Username:  client.Username,
		SeatIndex: seat,
		IsBot:     client.IsBot,
	}
	client.SetRoom(p.RoomID)

	if !client.IsBot && room.WaitingSince == nil {
		now := time.Now()
		room.WaitingSince = &now
	}

	data, _ := ws.NewMessage(ws.MsgRoomUpdate, room.ToInfo())
	e.hub.BroadcastToRoomHeld(room, data)
	room.Unlock()
}

func (e *Engine) handleLeaveRoom(client *ws.Client) {
	roomID := client.GetRoom()
	if roomID == 0 {
		return
	}
	client.SetRoom(0)
	e.hub.HandlePlayerLeave(client, roomID)
}

func (e *Engine) handleReady(client *ws.Client) {
	roomID := client.GetRoom()
	room := e.hub.GetRoom(roomID)
	if room == nil {
		return
	}

	room.Lock()

	if room.Phase != models.PhaseLobby {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("game already in progress")
		return
	}

	idx, player := room.FindPlayerByUserID(client.UserID)
	if idx < 0 {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("you are not a player in this room")
		return
	}

	player.IsReady = !player.IsReady

	data, _ := ws.NewMessage(ws.MsgRoomUpdate, room.ToInfo())
	e.hub.BroadcastToRoomHeld(room, data)

	if room.AllPlayersReady() {
		e.startGame(room)
	}

	room.Unlock()
}

func (e *Engine) startGame(room *models.Room) {
	room.Phase = models.PhaseDealing
	room.WaitingSince = nil
	hands := models.DealCards()

	firstPlayer := -1
	threeSpades := models.ThreeOfSpades()

	for i := 0; i < 4; i++ {
		room.Players[i].Hand = hands[i]
		room.Players[i].CardCount = 13
		room.Players[i].IsReady = false
		if models.ContainsCard(hands[i], threeSpades) {
			firstPlayer = i
		}
	}

	room.CurrentTurn = firstPlayer
	room.TablePlay = nil
	room.PassCount = 0
	room.Winner = -1
	room.Phase = models.PhasePlaying

	for i := 0; i < 4; i++ {
		p := room.Players[i]
		state := e.buildGameStateForPlayer(room, i)
		data, _ := ws.NewMessage(ws.MsgCardDealt, state)
		e.hub.SendToClient(p.UserID, data)
	}

	for _, s := range room.Spectators {
		state := e.buildGameStateForPlayer(room, -1)
		data, _ := ws.NewMessage(ws.MsgGameState, state)
		e.hub.SendToClient(s.UserID, data)
	}

	e.startTurnTimer(room)
	log.Printf("game started in room %d, first player: seat %d", room.ID, firstPlayer)
}

func (e *Engine) handlePlayCards(client *ws.Client, payload json.RawMessage) {
	var p ws.PlayCardsPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.Send <- ws.NewErrorMessage("invalid play_cards payload")
		return
	}

	roomID := client.GetRoom()
	room := e.hub.GetRoom(roomID)
	if room == nil {
		return
	}

	room.Lock()

	if room.Phase != models.PhasePlaying {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("game is not in playing phase")
		return
	}

	idx, player := room.FindPlayerByUserID(client.UserID)
	if idx < 0 || idx != room.CurrentTurn {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("not your turn")
		return
	}

	cards := make([]models.Card, len(p.Cards))
	for i, cp := range p.Cards {
		rank, err := models.ParseRank(cp.Rank)
		if err != nil {
			room.Unlock()
			client.Send <- ws.NewErrorMessage("invalid card rank: " + cp.Rank)
			return
		}
		suit, err := models.ParseSuit(cp.Suit)
		if err != nil {
			room.Unlock()
			client.Send <- ws.NewErrorMessage("invalid card suit: " + cp.Suit)
			return
		}
		cards[i] = models.Card{Rank: rank, Suit: suit}
	}

	if !PlayerOwnsCards(player.Hand, cards) {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("you don't have those cards")
		return
	}

	comboType, valid := ClassifyCombination(cards)
	if !valid {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("invalid card combination")
		return
	}

	if !CanBeat(room.TablePlay, cards, comboType) {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("your cards cannot beat the current play")
		return
	}

	player.Hand = models.RemoveCards(player.Hand, cards)
	player.CardCount = len(player.Hand)

	room.TablePlay = &models.TablePlay{
		PlayerIndex: idx,
		Cards:       cards,
		ComboType:   comboType,
	}
	room.PassCount = 0

	e.cancelTurnTimer(room.ID)

	moveData, _ := ws.NewMessage(ws.MsgMovePlayed, map[string]interface{}{
		"player_index": idx,
		"cards":        cards,
		"combo_type":   comboType,
	})
	e.hub.BroadcastToRoomHeld(room, moveData)

	if player.CardCount == 0 {
		e.endGame(room)
		room.Unlock()
		return
	}

	e.advanceTurn(room)
	room.Unlock()
}

func (e *Engine) handlePassTurn(client *ws.Client) {
	roomID := client.GetRoom()
	room := e.hub.GetRoom(roomID)
	if room == nil {
		return
	}

	room.Lock()

	if room.Phase != models.PhasePlaying {
		room.Unlock()
		return
	}

	idx, _ := room.FindPlayerByUserID(client.UserID)
	if idx < 0 || idx != room.CurrentTurn {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("not your turn")
		return
	}

	if room.TablePlay == nil {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("you must play cards to start the round")
		return
	}
	if room.TablePlay.PlayerIndex == idx {
		room.Unlock()
		client.Send <- ws.NewErrorMessage("you cannot pass on your own play")
		return
	}

	e.cancelTurnTimer(room.ID)
	room.PassCount++

	passData, _ := ws.NewMessage(ws.MsgTurnChange, map[string]interface{}{
		"action":       "pass",
		"player_index": idx,
	})
	e.hub.BroadcastToRoomHeld(room, passData)

	if room.PassCount >= 3 {
		room.TablePlay = nil
		room.PassCount = 0
	}

	e.advanceTurn(room)
	room.Unlock()
}

// advanceTurn must be called while room lock is held
func (e *Engine) advanceTurn(room *models.Room) {
	next := (room.CurrentTurn + 1) % 4

	attempts := 0
	for attempts < 4 {
		if room.Players[next] != nil && room.Players[next].CardCount > 0 {
			break
		}
		next = (next + 1) % 4
		attempts++
	}

	if room.TablePlay != nil && room.TablePlay.PlayerIndex == next {
		room.TablePlay = nil
		room.PassCount = 0
	}

	room.CurrentTurn = next

	turnData, _ := ws.NewMessage(ws.MsgTurnChange, map[string]interface{}{
		"current_turn": next,
		"table_clear":  room.TablePlay == nil,
	})
	e.hub.BroadcastToRoomHeld(room, turnData)

	e.startTurnTimer(room)
}

// countTwos returns how many 2s are in the hand.
func countTwos(hand []models.Card) int {
	count := 0
	for _, c := range hand {
		if c.Rank == models.Two {
			count++
		}
	}
	return count
}

// deadPigMultiplier returns the penalty multiplier for a loser.
// Extended rules: holding any 2 = 2x, 13 cards (never played) = 3x, all four 2s = 4x.
// Highest applicable multiplier wins (they don't stack).
func deadPigMultiplier(hand []models.Card, cardCount int) int {
	twos := countTwos(hand)
	if twos == 4 {
		return 4
	}
	if cardCount == 13 {
		return 3
	}
	if twos > 0 {
		return 2
	}
	return 1
}

// endGame must be called while room lock is held
func (e *Engine) endGame(room *models.Room) {
	winnerIdx := -1
	for i, p := range room.Players {
		if p != nil && p.CardCount == 0 {
			winnerIdx = i
			break
		}
	}

	e.cancelTurnTimer(room.ID)
	room.Phase = models.PhaseSettlement
	room.Winner = winnerIdx

	ante := room.AnteAmount
	settlement := make(map[string]interface{})
	results := make([]map[string]interface{}, 4)

	totalPot := 0
	for i := 0; i < 4; i++ {
		p := room.Players[i]
		if p == nil {
			continue
		}
		if i == winnerIdx {
			continue
		}
		multiplier := deadPigMultiplier(p.Hand, p.CardCount)
		loserPays := ante * multiplier
		totalPot += loserPays

		results[i] = map[string]interface{}{
			"seat":               i,
			"user_id":            p.UserID,
			"username":           p.Username,
			"cards_left":         p.CardCount,
			"twos_held":          countTwos(p.Hand),
			"penalty_multiplier": multiplier,
			"gold_delta":         -loserPays,
			"is_bot":             p.IsBot,
		}
	}

	serverFee := totalPot / 10
	winnerReceives := totalPot - serverFee

	if wp := room.Players[winnerIdx]; wp != nil {
		results[winnerIdx] = map[string]interface{}{
			"seat":               winnerIdx,
			"user_id":            wp.UserID,
			"username":           wp.Username,
			"cards_left":         0,
			"twos_held":          0,
			"penalty_multiplier": 0,
			"gold_delta":         winnerReceives,
			"is_bot":             wp.IsBot,
		}
	}

	settlement["winner"] = winnerIdx
	settlement["results"] = results
	settlement["server_fee"] = serverFee
	settlement["total_pot"] = totalPot

	data, _ := ws.NewMessage(ws.MsgSettlement, settlement)
	e.hub.BroadcastToRoomHeld(room, data)

	log.Printf("room %d settlement: winner=seat%d pot=%d fee=%d payout=%d",
		room.ID, winnerIdx, totalPot, serverFee, winnerReceives)

	go func(roomID int) {
		time.Sleep(5 * time.Second)
		r := e.hub.GetRoom(roomID)
		if r == nil {
			return
		}
		r.Lock()
		r.Phase = models.PhaseLobby
		r.TablePlay = nil
		r.PassCount = 0
		r.Winner = -1
		for _, p := range r.Players {
			if p != nil {
				p.Hand = nil
				p.CardCount = 0
				p.IsReady = false
			}
		}
		resetData, _ := ws.NewMessage(ws.MsgRoomUpdate, r.ToInfo())
		e.hub.BroadcastToRoomHeld(r, resetData)
		r.Unlock()
	}(room.ID)
}

func (e *Engine) handleChat(client *ws.Client, payload json.RawMessage) {
	var p ws.ChatPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}

	roomID := client.GetRoom()
	if roomID == 0 {
		return
	}

	data, _ := ws.NewMessage(ws.MsgChatRelay, ws.ChatPayload{
		Message: p.Message,
		Sender:  client.Username,
	})
	// Chat doesn't hold room lock, safe to use regular broadcast
	e.hub.BroadcastToRoom(roomID, data)
}

func (e *Engine) handleAutoMatch(client *ws.Client, payload json.RawMessage) {
	var p ws.AutoMatchPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.Send <- ws.NewErrorMessage("invalid auto_match payload")
		return
	}
	if e.mm != nil {
		e.mm.RequestMatch(client, p.AnteLevel)
	}
}

func (e *Engine) startTurnTimer(room *models.Room) {
	e.cancelTurnTimer(room.ID)

	roomID := room.ID
	turnSeat := room.CurrentTurn

	timer := time.AfterFunc(TurnTimeout, func() {
		r := e.hub.GetRoom(roomID)
		if r == nil {
			return
		}
		r.Lock()

		if r.Phase != models.PhasePlaying || r.CurrentTurn != turnSeat {
			r.Unlock()
			return
		}

		r.PassCount++
		timeoutData, _ := ws.NewMessage(ws.MsgTurnChange, map[string]interface{}{
			"action":       "timeout",
			"player_index": turnSeat,
		})
		e.hub.BroadcastToRoomHeld(r, timeoutData)

		if r.PassCount >= 3 {
			r.TablePlay = nil
			r.PassCount = 0
		}
		e.advanceTurn(r)
		r.Unlock()
	})

	e.turnTimers[room.ID] = timer
}

func (e *Engine) cancelTurnTimer(roomID int) {
	if timer, ok := e.turnTimers[roomID]; ok {
		timer.Stop()
		delete(e.turnTimers, roomID)
	}
}

type GameStatePayload struct {
	RoomID      int               `json:"room_id"`
	Phase       models.GamePhase  `json:"phase"`
	CurrentTurn int               `json:"current_turn"`
	Hand        []models.Card     `json:"hand,omitempty"`
	Players     []PlayerInfo      `json:"players"`
	TablePlay   *models.TablePlay `json:"table_play"`
	AnteAmount  int               `json:"ante_amount"`
}

type PlayerInfo struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	CardCount int    `json:"card_count"`
	SeatIndex int    `json:"seat_index"`
	IsReady   bool   `json:"is_ready"`
	IsBot     bool   `json:"is_bot"`
}

func (e *Engine) buildGameStateForPlayer(room *models.Room, seatIdx int) GameStatePayload {
	state := GameStatePayload{
		RoomID:      room.ID,
		Phase:       room.Phase,
		CurrentTurn: room.CurrentTurn,
		TablePlay:   room.TablePlay,
		AnteAmount:  room.AnteAmount,
		Players:     make([]PlayerInfo, 0, 4),
	}

	for _, p := range room.Players {
		if p == nil {
			continue
		}
		state.Players = append(state.Players, PlayerInfo{
			UserID:    p.UserID,
			Username:  p.Username,
			CardCount: p.CardCount,
			SeatIndex: p.SeatIndex,
			IsReady:   p.IsReady,
			IsBot:     p.IsBot,
		})
	}

	if seatIdx >= 0 && seatIdx < 4 && room.Players[seatIdx] != nil {
		state.Hand = room.Players[seatIdx].Hand
	}

	return state
}

func (e *Engine) buildRoomState(room *models.Room, seatIdx int) GameStatePayload {
	return e.buildGameStateForPlayer(room, seatIdx)
}
