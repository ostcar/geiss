package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ASGIHandler handels all incomming requests
func asgiHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	if websocket.IsWebSocketUpgrade(req) {
		if err = asgiWebsocketHandler(w, req); err != nil {
			log.Panic(err.Error())
		}
		return
	}

	err = asgiHTTPHandler(w, req)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Writes an output to the log for each incomming request.
func httpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
