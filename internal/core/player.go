package core

import (
	"github.com/google/uuid"
	"pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
)

func AddPlayer(name string) (*entities.Player, error) {
	id, _ := uuid.NewUUID()
	player := entities.Player{ID: id, Name: name}

	tx := database.Db.Create(&player)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &player, nil
}
