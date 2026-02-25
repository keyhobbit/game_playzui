package ws

import "encoding/json"

type MessageType string

const (
	// Client -> Server
	MsgJoinRoom  MessageType = "join_room"
	MsgLeaveRoom MessageType = "leave_room"
	MsgReady     MessageType = "ready"
	MsgPlayCards MessageType = "play_cards"
	MsgPassTurn  MessageType = "pass_turn"
	MsgChat      MessageType = "chat"
	MsgAutoMatch MessageType = "auto_match"

	// Server -> Client
	MsgRoomUpdate  MessageType = "room_update"
	MsgGameState   MessageType = "game_state"
	MsgCardDealt   MessageType = "card_dealt"
	MsgMovePlayed  MessageType = "move_played"
	MsgTurnChange  MessageType = "turn_change"
	MsgSettlement  MessageType = "settlement"
	MsgError       MessageType = "error"
	MsgChatRelay   MessageType = "chat_relay"
	MsgRoomList    MessageType = "room_list"
	MsgMatchFound  MessageType = "match_found"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type JoinRoomPayload struct {
	RoomID int `json:"room_id"`
}

type PlayCardsPayload struct {
	Cards []CardPayload `json:"cards"`
}

type CardPayload struct {
	Rank string `json:"rank"`
	Suit string `json:"suit"`
}

type ChatPayload struct {
	Message string `json:"message"`
	Sender  string `json:"sender,omitempty"`
}

type AutoMatchPayload struct {
	AnteLevel int `json:"ante_level"`
}

type ErrorPayload struct {
	Message string `json:"error"`
}

func NewMessage(msgType MessageType, payload interface{}) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := Message{Type: msgType, Payload: p}
	return json.Marshal(msg)
}

func NewErrorMessage(errMsg string) []byte {
	data, _ := NewMessage(MsgError, ErrorPayload{Message: errMsg})
	return data
}
