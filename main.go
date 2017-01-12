/*
This is an asgi protocol server and should be an alternative to daphne.
*/
package main

import (
	"fmt"
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
		cli.StringSliceFlag{
			Name:  "static, s",
			Value: nil,
			Usage: "url path prefix and file path to serve static file in the form /static/:/path/to/static/files",
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
			100)
		startHTTPServer(
			fmt.Sprintf("%s:%d", c.String("host"), c.Int64("port")),
			c.StringSlice("static"))
		return nil
	}
	app.Run(os.Args)

}
