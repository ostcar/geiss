/*
This is an asgi protocol server and should be an alternative to daphne.
*/
package main

import (
	"goasgiserver/asgi"
	"goasgiserver/asgi/redis"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var channelLayer asgi.ChannelLayer

func init() {
	channelLayer = redis.NewChannelLayer()
}

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

func main() {
	http.HandleFunc("/", asgiHandler)
	log.Fatal(http.ListenAndServe(":8000", httpLogger(http.DefaultServeMux)))
}
