package api

import (
	"flag"
	"github.com/gorilla/mux"
	"net/http"
)

var addr = flag.String("addr", "0.0.0.0:8080", "api service address")

func Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/ws", WsHandler)
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/config", ConfigHandler)
	http.Handle("/", r)
	_ = http.ListenAndServe(*addr, nil)
}
