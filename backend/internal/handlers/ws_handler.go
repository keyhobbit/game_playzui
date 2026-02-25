package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/game-playzui/tienlen-server/internal/auth"
	"github.com/game-playzui/tienlen-server/internal/matchmaking"
	"github.com/game-playzui/tienlen-server/internal/repository"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub        *ws.Hub
	jwtService *auth.JWTService
	userRepo   *repository.UserRepo
	mm         *matchmaking.Service
}

func NewWSHandler(hub *ws.Hub, jwtService *auth.JWTService, userRepo *repository.UserRepo, mm *matchmaking.Service) *WSHandler {
	return &WSHandler{
		hub:        hub,
		jwtService: jwtService,
		userRepo:   userRepo,
		mm:         mm,
	}
}

func (h *WSHandler) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, claims.UserID, claims.Username)
	h.hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}
