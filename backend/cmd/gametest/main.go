package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const serverURL = "http://localhost:8700"
const wsURL = "ws://localhost:8700"

type AuthResp struct {
	Token    string `json:"token"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

type WSMsg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Player struct {
	auth   AuthResp
	conn   *websocket.Conn
	hand   []map[string]string
	seat   int
	mu     sync.Mutex
}

var (
	players     [4]*Player
	currentTurn int
	gamePhase   string
	turnMu      sync.Mutex
	gameStarted = make(chan struct{})
	gameOver    = make(chan struct{})
	startOnce   sync.Once
	overOnce    sync.Once
)

func login(username, password string) AuthResp {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(serverURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("login failed for %s: %v", username, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var auth AuthResp
	json.Unmarshal(data, &auth)
	return auth
}

func connectWS(p *Player) {
	u, _ := url.Parse(wsURL + "/ws")
	q := u.Query()
	q.Set("token", p.auth.Token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("ws connect failed for %s: %v", p.auth.Username, err)
	}
	p.conn = conn

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg WSMsg
			json.Unmarshal(message, &msg)
			handleMessage(p, msg)
		}
	}()
}

func send(p *Player, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	msg := WSMsg{Type: msgType, Payload: data}
	raw, _ := json.Marshal(msg)
	p.conn.WriteMessage(websocket.TextMessage, raw)
}

func handleMessage(p *Player, msg WSMsg) {
	switch msg.Type {
	case "card_dealt":
		var payload struct {
			Hand        []map[string]string `json:"hand"`
			Players     []json.RawMessage   `json:"players"`
			CurrentTurn int                 `json:"current_turn"`
			Phase       string              `json:"phase"`
		}
		json.Unmarshal(msg.Payload, &payload)

		p.mu.Lock()
		p.hand = payload.Hand
		p.mu.Unlock()

		turnMu.Lock()
		currentTurn = payload.CurrentTurn
		gamePhase = payload.Phase
		turnMu.Unlock()

		cards := make([]string, len(payload.Hand))
		for i, c := range payload.Hand {
			cards[i] = c["rank"] + c["suit"]
		}
		fmt.Printf("  [%s] Dealt %d cards: %v\n", p.auth.Username, len(payload.Hand), cards)
		startOnce.Do(func() { close(gameStarted) })

	case "room_update":
		var payload struct {
			PlayerCount int    `json:"player_count"`
			Phase       string `json:"phase"`
			Name        string `json:"name"`
		}
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("  [%s] Room: %s %d/4 players, phase=%s\n", p.auth.Username, payload.Name, payload.PlayerCount, payload.Phase)

	case "move_played":
		var payload struct {
			PlayerIndex int                 `json:"player_index"`
			Cards       []map[string]string `json:"cards"`
		}
		json.Unmarshal(msg.Payload, &payload)
		cards := make([]string, len(payload.Cards))
		for i, c := range payload.Cards {
			cards[i] = c["rank"] + c["suit"]
		}
		fmt.Printf("  [%s] Seat %d played: %v\n", p.auth.Username, payload.PlayerIndex, cards)

	case "turn_change":
		var payload struct {
			CurrentTurn int    `json:"current_turn"`
			TableClear  bool   `json:"table_clear"`
			Action      string `json:"action"`
		}
		json.Unmarshal(msg.Payload, &payload)
		turnMu.Lock()
		currentTurn = payload.CurrentTurn
		turnMu.Unlock()

		extra := ""
		if payload.TableClear {
			extra = " (table cleared)"
		}
		if payload.Action != "" {
			extra += " [" + payload.Action + "]"
		}
		fmt.Printf("  [%s] Turn -> seat %d%s\n", p.auth.Username, payload.CurrentTurn, extra)

	case "settlement":
		var payload struct {
			Winner  int `json:"winner"`
			Results []struct {
				Username  string `json:"username"`
				GoldDelta int    `json:"gold_delta"`
				CardsLeft int    `json:"cards_left"`
			} `json:"results"`
		}
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("\n  === SETTLEMENT (winner: seat %d) ===\n", payload.Winner)
		for _, r := range payload.Results {
			prefix := ""
			if r.GoldDelta > 0 {
				prefix = "+"
			}
			marker := ""
			if r.CardsLeft == 0 {
				marker = " ** WINNER **"
			}
			fmt.Printf("  %s: %s%dG (cards left: %d)%s\n", r.Username, prefix, r.GoldDelta, r.CardsLeft, marker)
		}
		overOnce.Do(func() { close(gameOver) })

	case "error":
		var payload struct {
			Error string `json:"error"`
		}
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("  [%s] ERROR: %s\n", p.auth.Username, payload.Error)
	}
}

func main() {
	fmt.Println("=== TIEN LEN GAME FLOW TEST ===")
	fmt.Println()

	// 1. Login
	fmt.Println("1. Logging in 4 users...")
	for i := 0; i < 4; i++ {
		auth := login(fmt.Sprintf("user%d", i+1), "123321")
		players[i] = &Player{auth: auth, seat: i}
		fmt.Printf("   user%d: id=%d\n", i+1, auth.UserID)
	}

	// 2. Connect WebSockets
	fmt.Println("\n2. Connecting WebSockets...")
	for i := 0; i < 4; i++ {
		connectWS(players[i])
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("   All 4 connected")

	// 3. Join room 1
	fmt.Println("\n3. All join room 1...")
	for i := 0; i < 4; i++ {
		send(players[i], "join_room", map[string]int{"room_id": 1})
		time.Sleep(200 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)

	// 4. Ready up
	fmt.Println("\n4. All players ready up...")
	for i := 0; i < 4; i++ {
		send(players[i], "ready", map[string]string{})
		time.Sleep(100 * time.Millisecond)
	}

	// 5. Wait for game start
	fmt.Println("\n5. Waiting for cards...")
	select {
	case <-gameStarted:
		fmt.Println("   Cards dealt!")
	case <-time.After(5 * time.Second):
		fmt.Println("   TIMEOUT!")
		return
	}
	time.Sleep(500 * time.Millisecond)

	turnMu.Lock()
	fmt.Printf("\n6. Game started! First turn: seat %d\n", currentTurn)
	turnMu.Unlock()

	// 7. Play rounds - each player plays their lowest single card
	fmt.Print("\n7. Playing rounds (lowest single each turn)...\n\n")
	for round := 0; round < 52; round++ {
		select {
		case <-gameOver:
			goto done
		default:
		}

		turnMu.Lock()
		seat := currentTurn
		turnMu.Unlock()

		p := players[seat]
		p.mu.Lock()
		if len(p.hand) == 0 {
			p.mu.Unlock()
			time.Sleep(300 * time.Millisecond)
			continue
		}
		card := p.hand[0]
		p.hand = p.hand[1:]
		p.mu.Unlock()

		fmt.Printf("  Round %d: %s (seat %d) plays %s%s\n", round+1, p.auth.Username, seat, card["rank"], card["suit"])
		send(p, "play_cards", map[string]interface{}{"cards": []map[string]string{card}})
		time.Sleep(600 * time.Millisecond)

		// Other 3 players pass
		for pass := 0; pass < 3; pass++ {
			select {
			case <-gameOver:
				goto done
			default:
			}
			time.Sleep(300 * time.Millisecond)

			turnMu.Lock()
			nextSeat := currentTurn
			turnMu.Unlock()

			if nextSeat != seat {
				np := players[nextSeat]
				fmt.Printf("           %s (seat %d) passes\n", np.auth.Username, nextSeat)
				send(np, "pass_turn", map[string]string{})
				time.Sleep(300 * time.Millisecond)
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

done:
	select {
	case <-gameOver:
		fmt.Println("\n=== TEST COMPLETE: Game finished successfully! ===")
	case <-time.After(5 * time.Second):
		fmt.Println("\n=== Game still running (test ended) ===")
	}

	// Verify gold balances
	fmt.Println("\n8. Checking final gold balances...")
	for i := 0; i < 4; i++ {
		req, _ := http.NewRequest("GET", serverURL+"/api/user/profile", nil)
		req.Header.Set("Authorization", "Bearer "+players[i].auth.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var profile struct {
			Username    string `json:"username"`
			GoldBalance int64  `json:"gold_balance"`
		}
		json.Unmarshal(data, &profile)
		fmt.Printf("   %s: %d gold\n", profile.Username, profile.GoldBalance)
	}

	time.Sleep(1 * time.Second)
	for _, p := range players {
		if p.conn != nil {
			p.conn.Close()
		}
	}
	fmt.Println("\nDone.")
}
