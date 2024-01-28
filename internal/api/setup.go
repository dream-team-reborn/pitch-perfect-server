package api

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
)

//var addr = flag.String("addr", "0.0.0.0:8080", "api service address")

func Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/ws", WsHandler)
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/config", ConfigHandler)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("pitch-perfect.mstefanini.com"),
		Cache:      autocert.DirCache("./certs"), //Folder for storing certificates
	}

	server := &http.Server{
		Addr:    ":https",
		Handler: r,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	server.ListenAndServeTLS("", "") //Key and cert are coming from Let's Encrypt
}
