package matchmaking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/game-playzui/tienlen-server/internal/models"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

type MatchRequest struct {
	Client    *ws.Client
	AnteLevel int
}

type Service struct {
	rdb       *redis.Client
	hub       *ws.Hub
	queue     chan MatchRequest
	waitLists map[int][]MatchRequest // ante -> waiting clients
	mu        sync.Mutex
}

func NewService(rdb *redis.Client, hub *ws.Hub) *Service {
	return &Service{
		rdb:       rdb,
		hub:       hub,
		queue:     make(chan MatchRequest, 100),
		waitLists: make(map[int][]MatchRequest),
	}
}

func (s *Service) Start() {
	log.Println("matchmaking service started")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case req := <-s.queue:
			s.addToWaitList(req)
		case <-ticker.C:
			s.processWaitLists()
		}
	}
}

func (s *Service) RequestMatch(client *ws.Client, anteLevel int) {
	validAntes := map[int]bool{100: true, 500: true, 1000: true}
	if !validAntes[anteLevel] {
		client.Send <- ws.NewErrorMessage("invalid ante level, must be 100, 500, or 1000")
		return
	}
	s.queue <- MatchRequest{Client: client, AnteLevel: anteLevel}
}

func (s *Service) addToWaitList(req MatchRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitLists[req.AnteLevel] = append(s.waitLists[req.AnteLevel], req)
}

func (s *Service) processWaitLists() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ante, waiters := range s.waitLists {
		if len(waiters) == 0 {
			continue
		}

		// Try to find a room with available seats for this ante level
		room := s.findAvailableRoom(ante)
		if room == nil {
			continue
		}

		// Place waiting players into the room
		remaining := make([]MatchRequest, 0)
		for _, w := range waiters {
			if w.Client.GetRoom() > 0 {
				continue
			}

			room.Lock()
			seat := room.FindEmptySeat()
			if seat < 0 {
				room.Unlock()
				remaining = append(remaining, w)
				room = s.findAvailableRoom(ante)
				if room == nil {
					remaining = append(remaining, waiters[len(waiters)-len(remaining):]...)
					break
				}
				continue
			}

			room.Players[seat] = &models.Player{
				UserID:    w.Client.UserID,
				Username:  w.Client.Username,
				SeatIndex: seat,
			}
			w.Client.SetRoom(room.ID)
			room.Unlock()

			data, _ := ws.NewMessage(ws.MsgMatchFound, map[string]interface{}{
				"room_id":   room.ID,
				"room_name": room.Name,
				"seat":      seat,
			})
			w.Client.Send <- data
		}

		s.waitLists[ante] = remaining
	}
}

func (s *Service) findAvailableRoom(ante int) *models.Room {
	for _, room := range s.hub.Rooms {
		room.RLock()
		if room.AnteAmount == ante && room.Phase == models.PhaseLobby && room.PlayerCount() < 4 {
			room.RUnlock()
			return room
		}
		room.RUnlock()
	}
	return nil
}

func (s *Service) FindAvailableRoom(ante int) *models.Room {
	return s.findAvailableRoom(ante)
}

// TrackRoomOccupancy updates Redis with room occupancy for monitoring
func (s *Service) TrackRoomOccupancy(roomID int, playerCount int) {
	ctx := context.Background()
	key := fmt.Sprintf("room:%d:occupancy", roomID)
	data, _ := json.Marshal(map[string]interface{}{
		"player_count": playerCount,
		"updated_at":   time.Now().Unix(),
	})
	s.rdb.Set(ctx, key, data, 5*time.Minute)
}
