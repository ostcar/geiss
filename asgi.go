package main

import (
	"fmt"
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

// Channel which is used to communicate between the functions globalReceive() and
// readFromChannel()
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
			if data.receiver != nil {
				// Got a new receiver for a channelname
				receivers[data.channelname] = data
			} else {
				// Else, delete an existing channelname. delete does nothing, if the
				// channelname does not exist.
				delete(receivers, data.channelname)
			}

		case message := <-globalMessage:
			// Got a global message
			receiver, ok := receivers[message.channelname]
			if !ok {
				// Noone is listening for this channel.
				log.Printf("Error: Got message on global channel without a receiver, %s", message.message)
				continue
			}
			// Send the message to the receiver.
			// Do not block. Only try to send the message for one second.
			go func(receiver globalReceiveData, m asgi.Message) {
				timeout := time.After(time.Second)
				select {
				case receiver.receiver <- m:
				case <-timeout:
					log.Printf(
						"Tried to send a message from %s to a receiver but it was not read. This should never happen. The message was %s",
						receiver.channelname,
						m,
					)
				}
			}(receiver, message.message)
		}
	}
}

// readFromChannel registers a asgi channelname. It returns two (go-)channels.
// The first one will send the messages received on the registered asgi channel
// The second should be closed by the caller to unregister the channel.
func readFromChannel(channelname string) (messages chan asgi.Message, done chan bool) {
	messages = make(chan asgi.Message)
	done = make(chan bool)
	go func() {
		// Wait until the done channel was closed
		<-done
		// then send a message to globalReceiveChan to remove the channel from the list
		// of receivers
		globalReceiveChan <- globalReceiveData{channelname: channelname}
	}()
	globalReceiveChan <- globalReceiveData{channelname: channelname, receiver: messages}
	return
}

// readTimeout reads from the given channel for Duration. Returns the received
// message. If timeout happens first, then returns an error
func readTimeout(c chan asgi.Message, t time.Duration) (m asgi.Message, err error) {
	timeout := time.After(t)
	select {
	case m = <-c:
	case <-timeout:
		err = fmt.Errorf("could not receive a message in time")
	}
	return
}
