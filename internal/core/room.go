package core

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	database "pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
	"sync"
)

var rooms map[uuid.UUID]chan string
var roomsMutex sync.Mutex

func CreateRoom(creatorId uuid.UUID, name string) (uuid.UUID, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Error().Msg("Impossible to create UUID")
	}

	creator, err := GetPlayer(creatorId)
	if err != nil {
		return uuid.Nil, err
	}

	var players []entities.Player
	players = append(players, creator)

	room := entities.Room{ID: id, Players: players}
	database.Db.Create(&room)

	var c chan string
	roomsMutex.Lock()
	if rooms == nil {
		rooms = make(map[uuid.UUID]chan string)
	}
	rooms[room.ID] = c
	roomsMutex.Unlock()

	go roomCycle(c)

	return room.ID, nil
}

func GetAllRooms() ([]entities.Room, error) {
	var rooms []entities.Room
	tx := database.Db.Preload("Players").Find(&rooms)
	return rooms, tx.Error
}

func GetChannelByRoom(roomId uuid.UUID) (*chan string, error) {
	roomsMutex.Lock()
	if rooms == nil {
		rooms = make(map[uuid.UUID]chan string)
	}
	c, ok := rooms[roomId]
	if ok {
		return &c, nil
	}
	roomsMutex.Unlock()
	return nil, fmt.Errorf("math: square root of negative number %s", roomId.String())
}

func roomCycle(c chan string) {
	for {
		msg := <-c
		log.Info().Msg(msg)
	}
}
