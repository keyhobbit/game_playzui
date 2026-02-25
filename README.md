# Tien Len Mien Nam - Real-time Multiplayer Card Game

A high-concurrency, real-time multiplayer card game (Tien Len Mien Nam / Southern Vietnamese Poker) for Android and iOS. Supports up to 1,000 game rooms, each accommodating 4 players and 3 spectators with real-time chat and betting.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.22+ (Goroutines, WebSockets) |
| Client | Godot Engine 4.x (GDScript, 2D) |
| Communication | WebSockets (JSON) + REST API |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Infrastructure | Docker & Docker Compose |

## Architecture

```
┌─────────────────────────────────────────────────┐
│                Docker Compose                    │
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │         Go Server (:8700)                 │   │
│  │  ┌──────────┐  ┌──────────────────────┐  │   │
│  │  │ REST API  │  │   WebSocket Hub      │  │   │
│  │  └────┬─────┘  │  ┌──────────────┐   │  │   │
│  │       │         │  │ Game Engine   │   │  │   │
│  │       │         │  │ Matchmaking   │   │  │   │
│  │       │         │  └──────────────┘   │  │   │
│  │       │         └─────────┬───────────┘  │   │
│  └───────┼───────────────────┼──────────────┘   │
│          │                   │                   │
│  ┌───────▼──────┐   ┌───────▼──────┐           │
│  │ PostgreSQL    │   │    Redis      │           │
│  │   (:8702)     │   │   (:8701)     │           │
│  └──────────────┘   └──────────────┘           │
└─────────────────────────────────────────────────┘
         ▲                    ▲
         │    REST + WS       │
    ┌────┴────────────────────┴────┐
    │     Godot 4.x Clients        │
    │   (Android / iOS / Desktop)  │
    └──────────────────────────────┘
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Godot Engine 4.2+ (for client development)
- Go 1.22+ (for backend development without Docker)

### Start the Backend

```bash
docker-compose up -d --build
```

Services will be available at:
- **Game Server**: `http://localhost:8700` (REST) / `ws://localhost:8700/ws` (WebSocket)
- **Health Check**: `http://localhost:8700/health`
- **Redis**: `localhost:8701`
- **PostgreSQL**: `localhost:8702`

### Run the Client

1. Open `client/project.godot` in Godot 4.x
2. Edit `client/scripts/autoload/NetworkConfig.gd` to set your server IP if not localhost
3. Press F5 to run, or export to Android APK

### Android Deployment

```bash
# Export APK from Godot Editor, then:
adb install -r tienlen.apk
```

## Project Structure

```
game_playzui/
├── backend/
│   ├── cmd/server/main.go          # Entry point
│   ├── internal/
│   │   ├── config/                 # Environment configuration
│   │   ├── models/                 # Card, User, Room data models
│   │   ├── auth/                   # JWT authentication & middleware
│   │   ├── handlers/               # REST & WebSocket HTTP handlers
│   │   ├── bot/                    # AI bot manager, player, strategy
│   │   ├── game/                   # Game engine & card validation
│   │   ├── matchmaking/            # Room allocation & auto-match
│   │   ├── ws/                     # WebSocket hub, client, messages
│   │   └── repository/             # Database & migration layer
│   ├── migrations/                 # SQL migration files
│   ├── Dockerfile
│   ├── go.mod & go.sum
├── client/
│   ├── project.godot               # Godot project config
│   ├── scenes/                     # Login, Lobby, Game Room scenes
│   ├── scripts/
│   │   ├── autoload/               # NetworkConfig, AuthManager, WebSocketClient
│   │   ├── login/                  # Login scene logic
│   │   ├── lobby/                  # Room list & matchmaking
│   │   └── game/                   # Game room logic & card rendering
│   ├── assets/                     # Card sprites & assets
│   └── export_presets.cfg          # Android/iOS export configs
├── docker-compose.yml
└── README.md
```

## REST API

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/auth/register` | No | Create account (returns JWT) |
| POST | `/api/auth/login` | No | Login (returns JWT) |
| GET | `/api/user/profile` | Yes | Get user profile & gold balance |
| GET | `/api/rooms` | Yes | List rooms (filter: `?ante=100`) |
| GET | `/health` | No | Health check |

## WebSocket Protocol

Connect to `ws://host:8700/ws?token=<JWT>`.

### Client -> Server Messages

```json
{"type": "join_room",   "payload": {"room_id": 5}}
{"type": "leave_room",  "payload": {}}
{"type": "ready",       "payload": {}}
{"type": "play_cards",  "payload": {"cards": [{"rank": "3", "suit": "S"}]}}
{"type": "pass_turn",   "payload": {}}
{"type": "chat",        "payload": {"message": "hello"}}
{"type": "auto_match",  "payload": {"ante_level": 100}}
```

### Server -> Client Messages

- `room_update` - Room state changed (players joined/left)
- `card_dealt` - Cards dealt to you (includes your hand)
- `game_state` - Full game state update
- `move_played` - A player played cards
- `turn_change` - Turn advanced to next player
- `settlement` - Game ended, gold distributed
- `chat_relay` - Chat message from another player
- `match_found` - Auto-match found a room
- `error` - Error message

## Game Rules (Tien Len Mien Nam)

### Card Ranking
- **Ranks** (low to high): 3, 4, 5, 6, 7, 8, 9, 10, J, Q, K, A, 2
- **Suits** (low to high): Spades, Clubs, Diamonds, Hearts

### Valid Combinations
- **Single**: Any one card
- **Pair**: Two cards of the same rank
- **Triple**: Three cards of the same rank
- **Sequence**: 3+ consecutive singles (no 2s allowed)
- **Double Sequence**: 3+ consecutive pairs (no 2s, minimum 6 cards)
- **Four of a Kind**: All four cards of the same rank

### Special Rules
- Player holding 3 of Spades goes first
- Must play same combination type with higher value to beat
- "Chop Pig": Four-of-a-kind or double sequence (6+ cards) beats a single 2
- Double sequence (8+ cards) beats a pair of 2s
- 30-second turn timer (auto-pass on timeout)
- 3 consecutive passes = round cleared, last player starts new round

### Betting & Settlement
- Fixed ante rooms: 100G, 500G, 1000G
- **Dead Pig penalties** (highest applicable multiplier):
  - Holding any 2 at game end: **2x** penalty
  - Holding all 13 cards (never played): **3x** penalty
  - Holding all four 2s: **4x** penalty
- **Server fee**: 10% of total pot deducted; winner receives 90%
- Example: 3 losers pay 100G each (no dead pig) = 300G pot, 30G fee, winner gets 270G

### AI Bots
- 30 dedicated bot rooms (10 per ante level) with 3 bots each, waiting for a human player
- Bots auto-fill regular rooms after 30 seconds if humans are waiting
- Three difficulty tiers: Easy (random), Medium (minimum winning play), Hard (strategic with 2s conservation)
- Bots play with 1-3 second delays to feel human-like

## Testing on Phone (Web Client)

The easiest way to test on your Android phone without building an APK:

```bash
# 1. Start the backend
docker-compose up -d --build

# 2. Serve the web client from the project root
cd /path/to/game_playzui
python3 -m http.server 9000

# 3. Open on your phone (same WiFi network)
# http://<your-computer-ip>:9000/tools/test_client.html
# Set the Server IP:Port to <your-computer-ip>:8700
```

Features:
- Login/Register
- **Quick Play vs Bots** buttons (100G, 500G, 1000G) - one tap to join a bot room and auto-ready
- Full card game UI with Kenney card assets
- Settlement overlay showing dead pig penalties and server fee
- Bot players marked with purple "BOT" badge

## Development

### Backend (without Docker)

```bash
cd backend
go run ./cmd/server

# Environment variables:
# DB_HOST=localhost DB_PORT=5432 DB_USER=tienlen DB_PASSWORD=tienlen_secret
# REDIS_ADDR=localhost:6379
# JWT_SECRET=your-secret
```

### Run Tests

```bash
cd backend
go test ./...
```

### Build

```bash
cd backend
go build -o server ./cmd/server
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | 8700 | Server listen port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | tienlen | Database user |
| `DB_PASSWORD` | tienlen_secret | Database password |
| `DB_NAME` | tienlen | Database name |
| `REDIS_ADDR` | localhost:6379 | Redis address |
| `JWT_SECRET` | dev-secret-key | JWT signing secret |

## License

Private - All rights reserved.
