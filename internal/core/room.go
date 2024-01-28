package core

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jmcvetta/randutil"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/iter"
	"math/rand"
	database "pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
	"sync"
	"time"
)

var roomsIndex map[uuid.UUID]chan RoomCmd
var roomsMutex sync.Mutex
var deckWords []entities.Word
var deckPhrases []entities.Phrase
var categories []entities.Category
var phrase entities.Phrase
var hands map[uuid.UUID][]entities.Word
var trends map[uint]uint
var selectedCards map[uuid.UUID][]uint
var playersReview map[uuid.UUID]map[uuid.UUID]bool

const (
	Joined uint = iota
	Leave
	PlayerReady
	PlayerReadyTimeout
	PlayerCardSelected
	PlayerCardSelectedTimeout
	PlayerRatedOtherCards
	PlayerRatedOtherCardsTimeout
)

const (
	RoomStateWaiting uint = iota
	RoomStateTurnStarted
	RoomStateReview
)

type RoomCmd struct {
	Type     uint
	PlayerId uuid.UUID
	Player   entities.Player
	Cards    []uint
	Review   map[uuid.UUID]bool
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
			go roomCycle(*room, c)
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

	go roomCycle(room, c)

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

func LeaveRoom(leaverId uuid.UUID, roomId uuid.UUID) error {
	var room entities.Room
	tx := database.Db.Preload("Players").First(&room, roomId)
	if tx.Error != nil {
		return tx.Error
	}

	newPlayers := deleteElement(room.Players, leaverId)
	room.Players = newPlayers

	c, err := GetChannelByRoom(roomId)
	if err != nil {
		return err
	}

	*c <- RoomCmd{Type: Leave, PlayerId: leaverId}

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

func deleteElement(players []entities.Player, elem uuid.UUID) []entities.Player {
	result := make([]entities.Player, 0, len(players))
	for _, player := range players {
		if player.ID != elem {
			result = append(result, player)
		}
	}
	return result
}

func roomCycle(room entities.Room, c chan RoomCmd) {
	for {
		Cmd := <-c

		switch Cmd.Type {
		case Joined:
			joiner := Cmd.Player
			iter.ForEach(room.Players, func(player *entities.Player) {
				playersMutex.Lock()
				defer playersMutex.Unlock()

				chl, ok := playersIndex[player.ID]
				if ok {
					chl <- PlayerEvent{Type: RoomJoined, Player: joiner}
				}
			})

			newPlayers := append(room.Players, joiner)
			newPlayers, _ = uniqueSliceElements(newPlayers)
			room.Players = newPlayers
			break
		case Leave:
			leaver := Cmd.PlayerId
			newPlayers := deleteElement(room.Players, leaver)
			room.Players = newPlayers
			iter.ForEach(room.Players, func(player *entities.Player) {
				playersMutex.Lock()
				defer playersMutex.Unlock()

				chl, ok := playersIndex[player.ID]
				if ok {
					chl <- PlayerEvent{Type: RoomLeaved, PlayerId: leaver}
				}
			})
			break
		default:
			handleCmdDuringRoomState(Cmd, &room)
			break
		}
	}
}

func handleCmdDuringRoomState(cmd RoomCmd, room *entities.Room) {
	switch room.State {
	case RoomStateWaiting:
		handleCmdDuringWaiting(cmd, room)
		break
	case RoomStateTurnStarted:
		handleCmdDuringTurnStarted(cmd, room)
		break
	case RoomStateReview:
		handleCmdDuringReview(cmd, room)
		break
	}
}

func handleCmdDuringWaiting(cmd RoomCmd, room *entities.Room) {
	switch cmd.Type {
	case PlayerReady:
		newReadyPlayers := append(room.PlayersReady, cmd.PlayerId)
		newReadyPlayers, _ = uniqueSliceElements(newReadyPlayers)
		room.PlayersReady = newReadyPlayers
		if len(room.PlayersReady) >= len(room.Players) && len(room.Players) >= 1 {
			gameStart(room)
			startTurn(room)
		}
		break
	case PlayerReadyTimeout:
		break
	default:
		log.Error().Msg("Received a cmd not valid during waiting phase")
		break
	}
}

func handleCmdDuringTurnStarted(cmd RoomCmd, room *entities.Room) {
	switch cmd.Type {
	case PlayerCardSelected:
		selectedCards[cmd.PlayerId] = cmd.Cards
		if len(selectedCards) >= len(room.Players) {
			room.State += 1
			database.Db.Save(&room)
		}
		break
	case PlayerCardSelectedTimeout:
		break
	default:
		log.Error().Msg("Received a cmd not valid during turn started phase")
		break
	}
}

func handleCmdDuringReview(cmd RoomCmd, room *entities.Room) {
	switch cmd.Type {
	case PlayerRatedOtherCards:
		playersReview[cmd.PlayerId] = cmd.Review
		if len(playersReview) >= len(room.Players) {
			room.State += 1
			database.Db.Save(&room)
		}
		break
	case PlayerRatedOtherCardsTimeout:
		break
	default:
		log.Error().Msg("Received a cmd not valid during review phase")
		break
	}
}

func shuffleDeck[T comparable](deck *[]T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(*deck), func(i, j int) { (*deck)[i], (*deck)[j] = (*deck)[j], (*deck)[i] })
}

func generateTrends() {
	matrix := make([][]randutil.Choice, 5)
	matrix[0] = make([]randutil.Choice, 5)
	matrix[0][0] = randutil.Choice{Weight: 30, Item: 0}
	matrix[0][1] = randutil.Choice{Weight: 25, Item: 1}
	matrix[0][2] = randutil.Choice{Weight: 20, Item: 2}
	matrix[0][3] = randutil.Choice{Weight: 15, Item: 3}
	matrix[0][4] = randutil.Choice{Weight: 10, Item: 4}

	matrix[1] = make([]randutil.Choice, 5)
	matrix[1][0] = randutil.Choice{Weight: 10, Item: 0}
	matrix[1][1] = randutil.Choice{Weight: 30, Item: 1}
	matrix[1][2] = randutil.Choice{Weight: 25, Item: 2}
	matrix[1][3] = randutil.Choice{Weight: 20, Item: 3}
	matrix[1][4] = randutil.Choice{Weight: 15, Item: 4}

	matrix[2] = make([]randutil.Choice, 5)
	matrix[2][0] = randutil.Choice{Weight: 15, Item: 0}
	matrix[2][1] = randutil.Choice{Weight: 10, Item: 1}
	matrix[2][2] = randutil.Choice{Weight: 30, Item: 2}
	matrix[2][3] = randutil.Choice{Weight: 25, Item: 3}
	matrix[2][4] = randutil.Choice{Weight: 20, Item: 4}

	matrix[3] = make([]randutil.Choice, 5)
	matrix[3][0] = randutil.Choice{Weight: 20, Item: 0}
	matrix[3][1] = randutil.Choice{Weight: 15, Item: 1}
	matrix[3][2] = randutil.Choice{Weight: 10, Item: 2}
	matrix[3][3] = randutil.Choice{Weight: 30, Item: 3}
	matrix[3][4] = randutil.Choice{Weight: 25, Item: 4}

	matrix[4] = make([]randutil.Choice, 5)
	matrix[4][0] = randutil.Choice{Weight: 25, Item: 0}
	matrix[4][1] = randutil.Choice{Weight: 20, Item: 1}
	matrix[4][2] = randutil.Choice{Weight: 15, Item: 2}
	matrix[4][3] = randutil.Choice{Weight: 10, Item: 3}
	matrix[4][4] = randutil.Choice{Weight: 30, Item: 4}

	if trends == nil {
		oneRnd := rand.Intn(5)
		twoRnd := rand.Intn(5)
		threeRnd := rand.Intn(5)
		fourRnd := rand.Intn(5)
		fiveRnd := rand.Intn(5)
		trends = make(map[uint]uint)
		trends[1] = uint(oneRnd)
		trends[2] = uint(twoRnd)
		trends[3] = uint(threeRnd)
		trends[4] = uint(fourRnd)
		trends[5] = uint(fiveRnd)
	} else {
		for key, value := range trends {
			choice, _ := randutil.WeightedChoice(matrix[value])
			trends[key] = uint(choice.Item.(int))
		}
	}
}

func generateHands(room *entities.Room) {
	if hands == nil {
		hands = make(map[uuid.UUID][]entities.Word)
	}
	for _, v := range room.Players {
		hand, ok := hands[v.ID]
		if !ok || hand == nil {
			hand = make([]entities.Word, 0)
		}

		missingCard := 4 - len(hand)
		for i := 0; i < missingCard; i++ {
			hand = append(hand, deckWords[0])
			deckWords = deckWords[1 : len(deckWords)-1]
		}

		hands[v.ID] = hand
	}
}

func generatePhrase() {
	phrase = deckPhrases[0]
	deckPhrases = deckPhrases[1 : len(deckPhrases)-1]
}

func resetInternal() {
	selectedCards = make(map[uuid.UUID][]uint)
	playersReview = make(map[uuid.UUID]map[uuid.UUID]bool)
}

func getReviewWinner(room *entities.Room) uuid.UUID {
	reviewCount := make(map[uuid.UUID]uint)
	for _, p := range room.Players {
		reviewCount[p.ID] = 0
	}

	for p, reviews := range playersReview {
		for id, liked := range reviews {
			if liked && p != id {
				reviewCount[id] += 1
			}
		}
	}

	var m uint
	var winner uuid.UUID
	for p, c := range reviewCount {
		if c > m {
			m = c
			winner = p
		}
	}

	return winner
}

func gameStart(room *entities.Room) {
	room.State += 1
	database.Db.Save(&room)
	deckWords, _ = GetWords()
	deckPhrases, _ = GetPhrases()
	categories, _ = GetCategories()
	shuffleDeck(&deckWords)
	shuffleDeck(&deckPhrases)
	generateTrends()
	iter.ForEach(room.Players,
		func(player *entities.Player) {
			playersMutex.Lock()
			defer playersMutex.Unlock()
			c, ok := playersIndex[(*player).ID]
			if ok {
				c <- PlayerEvent{Type: GameStarted, Trends: trends}
			}
		})
}

func startTurn(room *entities.Room) {
	generatePhrase()
	generateHands(room)
	resetInternal()
	iter.ForEach(room.Players,
		func(player *entities.Player) {
			hand, ok := hands[player.ID]
			if !ok {
				log.Error().Msg("Impossible to get player hand in state")
				return
			}

			playersMutex.Lock()
			defer playersMutex.Unlock()
			c, ok := playersIndex[(*player).ID]
			if ok {
				c <- PlayerEvent{Type: TurnStarted, Cards: hand, Phrase: phrase}
			}
		})
}

func endTurn(room *entities.Room) {
	generateTrends()
	_ = getReviewWinner(room)
}
