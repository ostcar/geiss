/*
This is an asgi protocol server and should be an alternative to daphne.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli"

	"github.com/ostcar/goasgiserver/asgi"
	"github.com/ostcar/goasgiserver/asgi/redis"
)

var channelLayer asgi.ChannelLayer

func init() {

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
		cli.IntFlag{
			Name:  "port, p",
			Value: 8000,
			Usage: "port to listen on",
		},
		cli.StringFlag{
			Name:  "redis, r",
			Value: ":6379",
			Usage: "host and port of the redis server in the form HOST:Port",
		},
		cli.StringFlag{
			Name:  "redis-prefix",
			Value: "asgi:",
			Usage: "prefix of the redis keys",
		},
		cli.IntFlag{
			Name:  "redis-capacity",
			Value: 100,
			Usage: "channel capacity",
		},
		cli.IntFlag{
			Name:  "redis-expiry",
			Value: 60,
			Usage: "seconds until a message to the redis channel layer will expire",
		},
	}
	app.Action = func(c *cli.Context) error {
		listen := fmt.Sprintf("%s:%d", c.String("host"), c.Int64("port"))
		channelLayer = redis.NewChannelLayer(
			c.Int("redis-expiry"),
			c.String("redis"),
			c.String("redis-prefix"),
			100)
		http.HandleFunc("/", asgiHandler)
		log.Printf("Start webserver to listen on %s", listen)
		log.Fatal(http.ListenAndServe(listen, httpLogger(http.DefaultServeMux)))
		return nil
	}
	app.Run(os.Args)

}
