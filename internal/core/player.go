package core

import (
	"github.com/google/uuid"
	"pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
	"sync"
)

var playersIndex map[uuid.UUID]chan PlayerEvent
var playersMutex sync.Mutex

const (
	RoomJoined uint = iota
	RoomLeave
)

type PlayerEvent struct {
	Type     uint
	RoomId   uuid.UUID
	PlayerId uuid.UUID
	Player   entities.Player
}

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

func AddPlayerConnection(id uuid.UUID) (*chan PlayerEvent, error) {
	playersMutex.Lock()
	if playersIndex == nil {
		playersIndex = make(map[uuid.UUID]chan PlayerEvent)
	}
	c, ok := playersIndex[id]
	if ok {
		return &c, nil
	}
	c = make(chan PlayerEvent)
	playersIndex[id] = c
	playersMutex.Unlock()
	return &c, nil
}
