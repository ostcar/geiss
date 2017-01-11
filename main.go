/*
This is an asgi protocol server and should be an alternative to daphne.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/urfave/cli"

	"github.com/ostcar/goasgiserver/asgi"
	"github.com/ostcar/goasgiserver/asgi/redis"
)

var channelLayer asgi.ChannelLayer

func init() {
	channelLayer = redis.NewChannelLayer(60, nil, "asgi:", 100)
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

func main() {
	app := cli.NewApp()
	app.Name = "goasgiserver"
	app.Usage = "an asgi protocol server"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host, H",
			Value: "localhost",
			Usage: "host to listen on",
		},
		cli.Int64Flag{
			Name:  "port, p",
			Value: 8000,
			Usage: "port to listen on",
		},
	}
	app.Action = func(c *cli.Context) error {
		listen := fmt.Sprintf("%s:%d", c.String("host"), c.Int64("port"))
		http.HandleFunc("/", asgiHandler)
		log.Printf("Start webserver to listen on %s", listen)
		log.Fatal(http.ListenAndServe(listen, httpLogger(http.DefaultServeMux)))
		return nil
	}
	app.Run(os.Args)

}
