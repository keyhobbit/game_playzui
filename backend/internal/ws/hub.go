package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/game-playzui/tienlen-server/internal/models"
)

type Hub struct {
	Clients    map[int64]*Client
	Rooms      map[int]*models.Room
	Register   chan *Client
	Unregister chan *Client
	Incoming   chan *ClientMessage
	mu         sync.RWMutex

	OnMessage func(client *Client, msg Message)
}

func (h *Hub) RegisterBotClient(client *Client) {
	h.mu.Lock()
	h.Clients[client.UserID] = client
	h.mu.Unlock()
	log.Printf("bot registered: user=%d username=%s", client.UserID, client.Username)
}

func (h *Hub) UnregisterBotClient(client *Client) {
	h.mu.Lock()
	if c, ok := h.Clients[client.UserID]; ok && c == client {
		delete(h.Clients, client.UserID)
	}
	h.mu.Unlock()
	log.Printf("bot unregistered: user=%d", client.UserID)
}

func (h *Hub) InjectMessage(client *Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.Incoming <- &ClientMessage{Client: client, Data: data}
}

func NewHub() *Hub {
	h := &Hub{
		Clients:    make(map[int64]*Client),
		Rooms:      make(map[int]*models.Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *ClientMessage, 256),
	}
	h.initRooms()
	return h
}

func (h *Hub) initRooms() {
	antes := []int{100, 500, 1000}
	roomID := 1
	for _, ante := range antes {
		count := 334
		if ante == 1000 {
			count = 332
		}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("Room %d (%dG)", roomID, ante)
			h.Rooms[roomID] = models.NewRoom(roomID, name, ante)
			roomID++
		}
	}
	log.Printf("initialized %d rooms", len(h.Rooms))
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if existing, ok := h.Clients[client.UserID]; ok {
				close(existing.Send)
			}
			h.Clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("client registered: user=%d username=%s", client.UserID, client.Username)

		case client := <-h.Unregister:
			h.mu.Lock()
			if c, ok := h.Clients[client.UserID]; ok && c == client {
				delete(h.Clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()

			roomID := client.GetRoom()
			if roomID > 0 {
				h.HandlePlayerLeave(client, roomID)
			}
			log.Printf("client unregistered: user=%d", client.UserID)

		case cm := <-h.Incoming:
			var msg Message
			if err := json.Unmarshal(cm.Data, &msg); err != nil {
				cm.Client.Send <- NewErrorMessage("invalid message format")
				continue
			}
			if h.OnMessage != nil {
				h.OnMessage(cm.Client, msg)
			}
		}
	}
}

func (h *Hub) GetRoom(id int) *models.Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Rooms[id]
}

func (h *Hub) GetClient(userID int64) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Clients[userID]
}

// BroadcastToRoom acquires a read lock on the room before sending.
// Do NOT call this while already holding the room lock — use BroadcastToRoomHeld instead.
func (h *Hub) BroadcastToRoom(roomID int, data []byte) {
	room := h.GetRoom(roomID)
	if room == nil {
		return
	}
	room.RLock()
	userIDs := h.collectRoomUserIDs(room)
	room.RUnlock()
	h.sendToUsers(userIDs, data)
}

// BroadcastToRoomHeld sends to all players/spectators in a room.
// Caller MUST already hold the room lock (read or write).
func (h *Hub) BroadcastToRoomHeld(room *models.Room, data []byte) {
	userIDs := h.collectRoomUserIDs(room)
	h.sendToUsers(userIDs, data)
}

func (h *Hub) collectRoomUserIDs(room *models.Room) []int64 {
	ids := make([]int64, 0, 4+len(room.Spectators))
	for _, p := range room.Players {
		if p != nil {
			ids = append(ids, p.UserID)
		}
	}
	for _, s := range room.Spectators {
		ids = append(ids, s.UserID)
	}
	return ids
}

func (h *Hub) sendToUsers(userIDs []int64, data []byte) {
	for _, uid := range userIDs {
		if c := h.GetClient(uid); c != nil {
			select {
			case c.Send <- data:
			default:
			}
		}
	}
}

func (h *Hub) SendToClient(userID int64, data []byte) {
	if c := h.GetClient(userID); c != nil {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *Hub) HandlePlayerLeave(client *Client, roomID int) {
	room := h.GetRoom(roomID)
	if room == nil {
		return
	}
	room.Lock()

	idx, _ := room.FindPlayerByUserID(client.UserID)
	if idx >= 0 {
		room.Players[idx] = nil
		if room.Phase != models.PhaseLobby {
			room.Phase = models.PhaseLobby
			room.TablePlay = nil
			room.PassCount = 0
			for _, p := range room.Players {
				if p != nil {
					p.IsReady = false
					p.Hand = nil
					p.CardCount = 0
				}
			}
		}
	} else {
		room.RemoveSpectator(client.UserID)
	}

	data, _ := NewMessage(MsgRoomUpdate, room.ToInfo())
	h.BroadcastToRoomHeld(room, data)
	room.Unlock()
}

func (h *Hub) ListRoomInfos() []models.RoomInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	infos := make([]models.RoomInfo, 0, len(h.Rooms))
	for _, room := range h.Rooms {
		room.RLock()
		infos = append(infos, room.ToInfo())
		room.RUnlock()
	}
	return infos
}
