package database

import (
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"pitch-perfect-server/internal/entities"
)

var Db gorm.DB

func Init() {
	db, err := gorm.Open(sqlite.Open("pitch-perfect-server.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(&entities.Player{}, &entities.Room{}, &entities.Category{}, &entities.Word{}, &entities.Phrase{})
	if err != nil {

		log.Error().Msg("Impossible to migrate tables")
	}

	Db = *db

	log.Info().Msg("DB Init finished")
}
