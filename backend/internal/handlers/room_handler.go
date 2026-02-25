package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/game-playzui/tienlen-server/internal/matchmaking"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

type RoomHandler struct {
	hub *ws.Hub
	mm  *matchmaking.Service
}

func NewRoomHandler(hub *ws.Hub, mm *matchmaking.Service) *RoomHandler {
	return &RoomHandler{hub: hub, mm: mm}
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	infos := h.hub.ListRoomInfos()

	// Optional filtering by ante
	if anteStr := r.URL.Query().Get("ante"); anteStr != "" {
		ante, err := strconv.Atoi(anteStr)
		if err == nil {
			filtered := infos[:0]
			for _, info := range infos {
				if info.AnteAmount == ante {
					filtered = append(filtered, info)
				}
			}
			infos = filtered
		}
	}

	// Only return rooms with activity or available seats by default
	if r.URL.Query().Get("all") != "true" {
		active := infos[:0]
		for _, info := range infos {
			if info.PlayerCount > 0 {
				active = append(active, info)
			}
		}
		// If no active rooms, show first 20 empty rooms
		if len(active) == 0 {
			limit := 20
			if len(infos) < limit {
				limit = len(infos)
			}
			active = infos[:limit]
		}
		infos = active
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rooms": infos,
		"total": len(infos),
	})
}
