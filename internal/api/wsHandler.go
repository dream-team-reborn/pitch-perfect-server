package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"pitch-perfect-server/internal/auth"
	"pitch-perfect-server/internal/core"
	"sync"
)

var ws = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }} // use default options

func WsHandler(w http.ResponseWriter, r *http.Request) {
	var mutex sync.Mutex
	socket, err := ws.Upgrade(w, r, nil)
	if err != nil {
		//log.Print("upgrade:", err)
		return
	}
	defer socket.Close()

	playerId, err := checkToken(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eventChannel, err := core.AddPlayerConnection(playerId)
	if err != nil {
		log.Error().AnErr("add_connection", err)
		return
	}

	var room uuid.UUID

	go listenEventChannel(eventChannel, socket, &mutex)

	for {
		mt, bytes, err := socket.ReadMessage()
		if err != nil {
			log.Err(err)
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(bytes, &msg); err != nil {
			panic(err)
		}

		log.Info().Interface("msg", msg).Send()

		msgType, ok := msg["Type"]
		if ok {
			response := make(map[string]interface{})
			response["Type"] = msgType

			switch msgType {
			case "CreateRoom":
				roomName, ok := msg["RoomName"].(string)
				if !ok {
					response["Error"] = "No room name"
					break
				}

				roomId, err := core.CreateRoom(roomName)
				if err != nil {
					response["Error"] = err
				}

				response["RoomId"] = roomId
				break

			case "GetRooms":
				rooms, err := core.GetAllRooms()
				if err != nil {
					log.Err(err)
					break
				}

				response["Rooms"] = rooms
				break

			case "JoinRoom":
				roomIdStr, ok := msg["RoomId"].(string)
				if !ok {
					response["Error"] = "No room name"
					break
				}

				roomId, err := uuid.Parse(roomIdStr)
				if err != nil {
					response["Error"] = err
					break
				}

				err = core.JoinRoom(playerId, roomId)
				if err == nil {
					room = roomId
				}

				response["Result"] = err == nil
				break

			case "LeaveRoom":
				roomIdStr, ok := msg["RoomId"].(string)
				if !ok {
					response["Error"] = "No room name"
					break
				}

				roomId, err := uuid.Parse(roomIdStr)
				if err != nil {
					response["Error"] = err
					break
				}

				err = core.LeaveRoom(playerId, roomId)
				response["Result"] = err == nil
				break

			case "PlayerReady":
				roomIdStr, ok := msg["RoomId"].(string)
				if !ok {
					response["Error"] = "No room id"
					break
				}

				roomId, err := uuid.Parse(roomIdStr)
				if err != nil {
					response["Error"] = err
					break
				}

				c, err := core.GetChannelByRoom(roomId)
				if err != nil {
					response["Error"] = err
					break
				}

				response = nil

				*c <- core.RoomCmd{Type: core.PlayerReady, PlayerId: playerId}
				break

			case "PlayerCardsSelected":
				roomIdStr, ok := msg["RoomId"].(string)
				if !ok {
					response["Error"] = "No room id"
					break
				}

				roomId, err := uuid.Parse(roomIdStr)
				if err != nil {
					response["Error"] = err
					break
				}

				cardsData, ok := msg["Cards"].([]interface{})
				if !ok {
					response["Error"] = "No player cards"
					break
				}
				cards := make([]uint, len(cardsData))
				for k, v := range cardsData {
					cards[k] = uint(v.(float64))
				}

				c, err := core.GetChannelByRoom(roomId)
				if err != nil {
					response["Error"] = err
					break
				}

				response = nil

				*c <- core.RoomCmd{Type: core.PlayerCardsSelected, PlayerId: playerId, Cards: cards}
				break

			case "PlayerRatedOtherCards":
				roomIdStr, ok := msg["RoomId"].(string)
				if !ok {
					response["Error"] = "No room id"
					break
				}

				roomId, err := uuid.Parse(roomIdStr)
				if err != nil {
					response["Error"] = err
					break
				}

				data, ok := msg["Reviews"]
				if !ok {
					response["Error"] = "No review name"
					break
				}
				reviews := make(map[uuid.UUID]bool)
				for k, v := range data.(map[string]interface{}) {
					id, err := uuid.Parse(k)
					if err != nil {
						response["Error"] = err
						break
					}

					reviews[id] = v.(bool)
				}

				c, err := core.GetChannelByRoom(roomId)
				if err != nil {
					response["Error"] = err
					break
				}

				response = nil

				*c <- core.RoomCmd{Type: core.PlayerRatedOtherCards, PlayerId: playerId, Reviews: reviews}
				break

			default:
				if err != nil {
					log.Err(err)
					break
				}
			}

			if response != nil {
				output, err := json.Marshal(response)
				if err != nil {
					log.Err(err)
					break
				}
				mutex.Lock()
				err = socket.WriteMessage(mt, output)
				mutex.Unlock()
				if err != nil {
					log.Err(err)
					break
				}
			}
		}
	}

	if room != uuid.Nil {
		_ = core.LeaveRoom(playerId, room)
	}

	*eventChannel <- core.PlayerEvent{Type: core.ConnectionDown}

	log.Warn().Msg("Conn destroyed")
}

func checkToken(r *http.Request) (uuid.UUID, error) {
	token := r.URL.Query().Get("token")
	return auth.CheckToken(token)
}

func listenEventChannel(c *chan core.PlayerEvent, socket *websocket.Conn, mt *sync.Mutex) {
	for {
		if socket == nil {
			break
		}

		if *c == nil {
			break
		}

		event := <-*c

		response := make(map[string]interface{})
		switch event.Type {
		case core.RoomJoined:
			response["Type"] = "RoomJoined"
			response["Player"] = event.Player
			break
		case core.RoomLeaved:
			response["Type"] = "RoomLeaved"
			response["PlayerId"] = event.PlayerId
			break
		case core.GameStarted:
			response["Type"] = "GameStarted"
			response["Trends"] = event.Trends
			break
		case core.TurnStarted:
			response["Type"] = "TurnStarted"
			response["Cards"] = event.Cards
			response["Phrase"] = event.Phrase
			break
		case core.AllPlayerSelectedCards:
			response["Type"] = "AllPlayerSelectedCards"
			response["PlayersCards"] = event.PlayersCards
			break
		case core.TurnEnded:
			response["Type"] = "TurnEnded"
			response["Trends"] = event.Trends
			response["Leaderboards"] = event.Leaderboards
			response["Result"] = event.Result
			response["LastTurn"] = event.LastTurn
			break
		case core.RoomCreated:
			response["Type"] = "RoomCreated"
			response["Room"] = event.Room
			break
		case core.ConnectionDown:
			return
		default:
			break
		}

		output, err := json.Marshal(response)
		if err != nil {
			log.Err(err)
		}

		mt.Lock()
		err = socket.WriteMessage(1, output)
		mt.Unlock()
		if err != nil {
			log.Err(err)
		}
	}
}
