package api

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", "0.0.0.0:8080", "api service address")

func Serve() {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/ws", WsHandler)
	http.HandleFunc("/login", LoginHandler)

	log.Fatal(http.ListenAndServe(*addr, nil))
}
