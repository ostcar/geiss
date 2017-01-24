/*
This is an asgi protocol server and should be an alternative to daphne.
*/
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/ostcar/geiss/asgi"
	"github.com/ostcar/geiss/asgi/redis"
)

var channelLayer asgi.ChannelLayer
var debug bool

func main() {
	app := cli.NewApp()
	app.Name = "geiss"
	app.Usage = "an asgi protocol server"
	app.HideHelp = true
	app.ArgsUsage = " " // If it is an empty string, then it shows a stupid default text
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
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "if set, sends error messages to the client",
			Destination: &debug,
		},
		cli.StringSliceFlag{
			Name:  "static, s",
			Value: nil,
			Usage: "url and file path to serve static files in the form /static/:/path/to/files",
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
		channelLayer = redis.NewChannelLayer(
			c.Int("redis-expiry"),
			c.String("redis"),
			c.String("redis-prefix"),
			c.Int("redis-capacity"))

		startHTTPServer(
			fmt.Sprintf("%s:%d", c.String("host"), c.Int64("port")),
			c.StringSlice("static"))
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Received an error: %s", err)
	}
}
