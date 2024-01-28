package api

import (
	"crypto/tls"
	"github.com/gorilla/handlers"
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

	credentials := handlers.AllowCredentials()
	methods := handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"})
	origins := handlers.AllowedOrigins([]string{"*"})
	headers := handlers.AllowedHeaders([]string{"Accept", "X-Access-Token", "X-Application-Name", "X-Request-Sent-Time", "Content-Type"})

	server := &http.Server{
		Addr:    ":https",
		Handler: handlers.CORS(credentials, methods, origins, headers)(r),
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	server.ListenAndServeTLS("", "") //Key and cert are coming from Let's Encrypt
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers:", "Origin, Content-Type, X-Auth-Token, Authorization")
	})
}
