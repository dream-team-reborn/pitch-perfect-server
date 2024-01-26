package main

import (
	"github.com/rs/zerolog"
	"pitch-perfect-server/internal/api"
	database "pitch-perfect-server/internal/db"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	database.Init()
	api.Serve()
}
