package main

import (
	"github.com/rs/zerolog"
	"pitch-perfect-server/internal/api"
	"pitch-perfect-server/internal/core"
	database "pitch-perfect-server/internal/db"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	database.Init()
	_ = core.InitConfig()
	_ = core.InitRooms()
	api.Serve()
}
