package api

import (
	"net/http"
)

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./config/1/game_configuration.json")
}
