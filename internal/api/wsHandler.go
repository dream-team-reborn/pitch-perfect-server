package api

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"pitch-perfect-server/internal/auth"
)

var ws = websocket.Upgrader{} // use default options

func WsHandler(w http.ResponseWriter, r *http.Request) {
	err := checkToken(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c, err := ws.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func checkToken(r *http.Request) error {
	token := r.Header.Get("Token")
	_, err := auth.CheckToken(token)
	return err
}
