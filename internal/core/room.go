package core

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/iter"
	database "pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
	"sync"
)

var roomsIndex map[uuid.UUID]chan RoomCmd
var roomsMutex sync.Mutex

const (
	Joined uint = iota
)

type RoomCmd struct {
	Type   uint
	Player entities.Player
}

func InitRooms() error {
	rooms, err := GetAllRooms()
	if err != nil {
		return err
	}

	iter.ForEach(rooms,
		func(room *entities.Room) {
			roomsMutex.Lock()
			defer roomsMutex.Unlock()
			if roomsIndex == nil {
				roomsIndex = make(map[uuid.UUID]chan RoomCmd)
			}
			c := make(chan RoomCmd)
			roomsIndex[room.ID] = c
			go roomCycle(c)
			return
		})

	return nil
}

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

	room := entities.Room{ID: id, Name: name, Players: players}
	database.Db.Create(&room)

	var c chan RoomCmd
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	if roomsIndex == nil {
		roomsIndex = make(map[uuid.UUID]chan RoomCmd)
	}
	roomsIndex[room.ID] = c

	go roomCycle(c)

	return room.ID, nil
}

func GetAllRooms() ([]entities.Room, error) {
	var rooms []entities.Room
	tx := database.Db.Preload("Players").Find(&rooms)
	return rooms, tx.Error
}

func JoinRoom(joinerId uuid.UUID, roomId uuid.UUID) error {
	var room entities.Room
	tx := database.Db.Preload("Players").First(&room, roomId)
	if tx.Error != nil {
		return tx.Error
	}

	player, err := GetPlayer(joinerId)
	if err != nil {
		return err
	}

	newPlayers := append(room.Players, player)
	newPlayers, onlyUnique := uniqueSliceElements(newPlayers)
	if !onlyUnique {
		room.Players = newPlayers
		tx = database.Db.Save(room)
		if err != nil {
			return err
		}
	}

	c, err := GetChannelByRoom(roomId)
	if err != nil {
		return err
	}

	*c <- RoomCmd{Type: Joined, Player: player}

	return nil
}

func GetChannelByRoom(roomId uuid.UUID) (*chan RoomCmd, error) {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()
	if roomsIndex == nil {
		roomsIndex = make(map[uuid.UUID]chan RoomCmd)
	}
	c, ok := roomsIndex[roomId]
	if ok {
		return &c, nil
	}
	return nil, fmt.Errorf("math: square root of negative number %s", roomId.String())
}

func uniqueSliceElements[T comparable](inputSlice []T) ([]T, bool) {
	onlyUnique := true
	uniqueSlice := make([]T, 0, len(inputSlice))
	seen := make(map[T]bool, len(inputSlice))
	for _, element := range inputSlice {
		if !seen[element] {
			uniqueSlice = append(uniqueSlice, element)
			seen[element] = true
		} else {
			onlyUnique = false
		}
	}
	return uniqueSlice, onlyUnique
}

func roomCycle(c chan RoomCmd) {
	for {
		Cmd := <-c
		jj, err := json.Marshal(Cmd)
		if err != nil {
			log.Err(err)
		}

		log.Info().Bytes("cmd", jj).Send()
	}
}
