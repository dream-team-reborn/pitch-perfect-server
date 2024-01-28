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
	RoomLeaved
	GameStarted
	TurnStarted
	AllPlayerSelectedCards
	TurnEnded
)

type PlayerEvent struct {
	Type         uint
	RoomId       uuid.UUID
	PlayerId     uuid.UUID
	Player       entities.Player
	Cards        []entities.Word
	Phrase       entities.Phrase
	PlayersCards map[uuid.UUID][]uint
	Players      map[uuid.UUID]uint
	Trends       map[uint]uint
	LastTurn     bool
	Leaderboards map[uuid.UUID]uint
	Result       map[uuid.UUID]uint
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
	defer playersMutex.Unlock()
	if playersIndex == nil {
		playersIndex = make(map[uuid.UUID]chan PlayerEvent)
	}
	c, ok := playersIndex[id]
	if ok {
		return &c, nil
	}
	c = make(chan PlayerEvent)
	playersIndex[id] = c
	return &c, nil
}
