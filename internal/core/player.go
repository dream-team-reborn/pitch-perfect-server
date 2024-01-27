package core

import (
	"github.com/google/uuid"
	"pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
	"sync"
)

var players map[uuid.UUID]chan string
var playersMutex sync.Mutex

func AddPlayer(name string) (entities.Player, error) {
	id, _ := uuid.NewUUID()
	player := entities.Player{ID: id, Name: name}
	tx := database.Db.Create(&player)
	return player, tx.Error
}

func GetPlayer(id uuid.UUID) (entities.Player, error) {
	var player entities.Player
	tx := database.Db.First(&player, id)
	return player, tx.Error
}

func AddPlayerConnection(id uuid.UUID) (*chan string, error) {
	playersMutex.Lock()
	if players == nil {
		players = make(map[uuid.UUID]chan string)
	}
	c, ok := players[id]
	if ok {
		return &c, nil
	}
	c = make(chan string)
	players[id] = c
	playersMutex.Unlock()
	return &c, nil
}
