package bot

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/game-playzui/tienlen-server/internal/models"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

var botIDCounter int64 = -1

func nextBotID() int64 {
	return atomic.AddInt64(&botIDCounter, -1)
}

var botNames = []string{
	"Bot_Alpha", "Bot_Beta", "Bot_Gamma", "Bot_Delta", "Bot_Epsilon",
	"Bot_Zeta", "Bot_Eta", "Bot_Theta", "Bot_Iota", "Bot_Kappa",
	"Bot_Lambda", "Bot_Mu", "Bot_Nu", "Bot_Xi", "Bot_Omicron",
	"Bot_Pi", "Bot_Rho", "Bot_Sigma", "Bot_Tau", "Bot_Upsilon",
	"Bot_Phi", "Bot_Chi", "Bot_Psi", "Bot_Omega",
	"Bot_Ace", "Bot_King", "Bot_Queen", "Bot_Jack", "Bot_Joker",
	"Bot_Shark", "Bot_Tiger", "Bot_Dragon", "Bot_Phoenix", "Bot_Storm",
	"Bot_Flash", "Bot_Blaze", "Bot_Frost", "Bot_Shadow", "Bot_Viper",
	"Bot_Hawk", "Bot_Wolf", "Bot_Bear", "Bot_Eagle", "Bot_Cobra",
	"Bot_Ninja", "Bot_Samurai", "Bot_Ronin", "Bot_Sensei", "Bot_Master",
}

const (
	DedicatedRoomsPerAnte = 10
	AutoFillInterval      = 30 * time.Second
	AutoFillWaitThreshold = 30 * time.Second
)

type Manager struct {
	hub     *ws.Hub
	bots    map[int64]*BotPlayer // botUserID -> BotPlayer
	mu      sync.Mutex
}

func NewManager(hub *ws.Hub) *Manager {
	return &Manager{
		hub:  hub,
		bots: make(map[int64]*BotPlayer),
	}
}

func (m *Manager) Run() {
	m.setupDedicatedRooms()

	ticker := time.NewTicker(AutoFillInterval)
	defer ticker.Stop()

	log.Println("bot manager started")
	for range ticker.C {
		m.autoFillRooms()
	}
}

func (m *Manager) setupDedicatedRooms() {
	antes := []int{100, 500, 1000}
	for _, ante := range antes {
		count := 0
		for roomID, room := range m.hub.Rooms {
			if count >= DedicatedRoomsPerAnte {
				break
			}
			room.RLock()
			isMatch := room.AnteAmount == ante && room.Phase == models.PhaseLobby && room.PlayerCount() == 0
			room.RUnlock()

			if !isMatch {
				continue
			}

			room.Lock()
			if room.PlayerCount() != 0 || room.Phase != models.PhaseLobby {
				room.Unlock()
				continue
			}
			room.HasBots = true
			room.Name = fmt.Sprintf("Bot Room %d (%dG)", roomID, ante)
			room.Unlock()

			m.addBotsToRoom(room, 3)
			count++
			log.Printf("set up dedicated bot room %d (%dG) with 3 bots", roomID, ante)
		}
	}
}

func (m *Manager) addBotsToRoom(room *models.Room, count int) {
	for i := 0; i < count; i++ {
		botID := nextBotID()
		name := botNames[rand.Intn(len(botNames))]
		diff := Difficulty(rand.Intn(3))

		client := ws.NewBotClient(m.hub, botID, name)
		m.hub.RegisterBotClient(client)

		room.Lock()
		seat := room.FindEmptySeat()
		if seat < 0 {
			room.Unlock()
			m.hub.UnregisterBotClient(client)
			break
		}
		room.Players[seat] = &models.Player{
			UserID:    botID,
			Username:  name,
			SeatIndex: seat,
			IsBot:     true,
		}
		client.SetRoom(room.ID)
		room.Unlock()

		bp := NewBotPlayer(client, room.ID, seat, diff)
		bp.Start()

		m.mu.Lock()
		m.bots[botID] = bp
		m.mu.Unlock()

		// Auto-ready the bot
		go func() {
			time.Sleep(500 * time.Millisecond)
			bp.sendReady()
		}()
	}
}

func (m *Manager) autoFillRooms() {
	now := time.Now()
	for _, room := range m.hub.Rooms {
		room.RLock()
		phase := room.Phase
		humanCount := room.HumanPlayerCount()
		playerCount := room.PlayerCount()
		hasBots := room.HasBots
		waitingSince := room.WaitingSince
		room.RUnlock()

		if phase != models.PhaseLobby || humanCount == 0 || playerCount >= 4 {
			continue
		}

		if hasBots {
			continue
		}

		if waitingSince == nil || now.Sub(*waitingSince) < AutoFillWaitThreshold {
			continue
		}

		botsNeeded := 4 - playerCount
		if botsNeeded <= 0 {
			continue
		}

		room.Lock()
		room.HasBots = true
		room.Unlock()

		log.Printf("auto-filling room %d with %d bots (human players waiting)", room.ID, botsNeeded)
		m.addBotsToRoom(room, botsNeeded)
	}
}
