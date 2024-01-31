package api

import (
	"crypto/tls"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
)

func Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/ws", WsHandler)
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/config", ConfigHandler)

	credentials := handlers.AllowCredentials()
	methods := handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"})
	origins := handlers.AllowedOrigins([]string{"*"})
	headers := handlers.AllowedHeaders([]string{"Accept", "X-Access-Token", "X-Application-Name", "X-Request-Sent-Time", "Content-Type"})
	handler := handlers.CORS(credentials, methods, origins, headers)(r)

	cert, err := tls.LoadX509KeyPair("./certs/domain.cert.pem", "./certs/private.key.pem")
	if err != nil {
		panic("Error in getting tls certs")
	}
	certs := make([]tls.Certificate, 0)
	certs = append(certs, cert)

	server := &http.Server{
		Addr:    ":https",
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: certs,
		},
	}

	server.ListenAndServeTLS("", "") //Key and cert are coming from Let's Encrypt
}
