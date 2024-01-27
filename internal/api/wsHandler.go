package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"pitch-perfect-server/internal/auth"
	"pitch-perfect-server/internal/core"
)

var ws = websocket.Upgrader{} // use default options

func WsHandler(w http.ResponseWriter, r *http.Request) {
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

	_, err = core.AddPlayerConnection(playerId)
	if err != nil {
		log.Error().AnErr("add_connection", err)
		return
	}

	for {
		mt, bytes, err := socket.ReadMessage()
		if err != nil {
			//log.Println("read:", err)
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(bytes, &msg); err != nil {
			panic(err)
		}

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

				roomId, err := core.CreateRoom(playerId, roomName)
				if err != nil {
					response["Error"] = err
				}

				response["RoomId"] = roomId

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

			default:
				if err != nil {
					log.Err(err)
					break
				}
			}

			output, err := json.Marshal(response)
			if err != nil {
				log.Err(err)
				break
			}
			err = socket.WriteMessage(mt, output)
			if err != nil {
				log.Err(err)
				break
			}
		}
	}

	log.Warn().Msg("Conn destroyed")
}

func checkToken(r *http.Request) (uuid.UUID, error) {
	token := r.URL.Query().Get("token")
	return auth.CheckToken(token)
}
