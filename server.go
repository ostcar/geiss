package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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
		handleError(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleError(w http.ResponseWriter, m string, status int) {
	log.Printf("Error: %s\n", m)
	if !debug {
		m = "Internal error."
	} else {
		m = fmt.Sprintf("%d: Error: %s.", status, m)
	}
	http.Error(w, m, status)
}

// Writes an output to the log for each incomming request.
func httpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func startHTTPServer(listen string, statics []string) {
	for _, static := range statics {
		paths := strings.SplitN(static, ":", 2)
		if len(paths) != 2 {
			log.Fatalf("Invalid argument for --static \"%s\"", static)
		}
		http.Handle(paths[0], http.StripPrefix(paths[0], http.FileServer(http.Dir(paths[1]))))
	}
	http.HandleFunc("/", asgiHandler)
	log.Printf("Start webserver to listen on %s", listen)
	log.Fatal(http.ListenAndServe(listen, httpLogger(http.DefaultServeMux)))
}
