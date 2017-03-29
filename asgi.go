package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/ostcar/geiss/asgi"
)

var globalChannelname string

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	globalChannelname = "geiss.response." + asgi.GetChannelnameRandom() + "!"
	globalReceiveChan = make(chan globalReceiveData)
}

type globalReceiveData struct {
	channelname string
	message     asgi.Message
	receiver    chan asgi.Message
}

var globalReceiveChan chan globalReceiveData

// globalReceive listens to the base asgi channel and dispaches the incomming
// messages to receivers.
func globalReceive() {
	globalMessage := make(chan globalReceiveData)
	receivers := make(map[string]globalReceiveData)

	// Read from the global channel.
	// Currently, this happens in one coroutine. It could be faster if there are
	// more coroutines or if the globalMessage channel has a buffer.
	go func() {
		for {
			channelname, message, err := channelLayer.Receive([]string{globalChannelname}, true)
			if err != nil {
				log.Printf("Error: Can not receive a message from the global channel: %s", err)
				continue
			}
			if channelname != "" {
				// Got a message.
				globalMessage <- globalReceiveData{channelname: channelname, message: message}
			}
		}
	}()

	for {
		select {
		case data := <-globalReceiveChan:
			// Someone wants to listen to a channel
			receivers[data.channelname] = data

		case message := <-globalMessage:
			// Got a global message
			receiver, ok := receivers[message.channelname]
			if !ok {
				// Noone is listening for this channel.
				log.Printf("Error: Got message on global channel without a receiver, %s", message.message)
				continue
			}
			// Send the message to the receiver.
			// This has to happen in a seperate function, so if the channel is closed,
			// we can handle the panic.
			// The receiver has to make sure to listen to the channel, in othercase,
			// this is a deadlock!
			func() {
				defer func() {
					if r := recover(); r != nil {
						// The channel was closed. So it is not needed anymore
						delete(receivers, message.channelname)
					}
				}()
				receiver.receiver <- message.message
			}()
		}
	}
}
