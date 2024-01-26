package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"pitch-perfect-server/internal/auth"
	"pitch-perfect-server/internal/core"
)

type LoginRequest struct {
	Name  string
	Token string
}

type LoginResponse struct {
	UserId string
	Token  string
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginRequest LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginRequest)

	if len(loginRequest.Token) == 0 {
		player, err := core.AddPlayer(loginRequest.Name)
		if err != nil {
			errorResponse(w)
			return
		}

		token, err := auth.GenerateToken(player.ID)
		if err != nil {
			errorResponse(w)
			return
		}

		okResponse(player.ID, token, w)
	}

	id, err := auth.CheckToken(loginRequest.Token)
	if err != nil {
		errorResponse(w)
		return
	}

	okResponse(id, loginRequest.Token, w)
}

func okResponse(playerId uuid.UUID, token string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	payload := LoginResponse{playerId.String(), token}
	err := json.NewEncoder(w).Encode(payload)
	if err != nil {
		return
	}
}

func errorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	return
}
