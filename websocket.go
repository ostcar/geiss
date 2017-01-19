package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ostcar/geiss/asgi"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type websocketMessage struct {
	// Type of the message. Can be websocket.TextMessage or websocket.BinaryMessage.
	Type int

	// Value to send through the websocket connection.
	Content []byte

	// Websocket Error. Should only set if the othre values are not set.
	Err *websocket.CloseError
}

// Read from a websocket connection and write any message to the read channel.
func readWebsocket(conn *websocket.Conn, read chan websocketMessage) {
	defer close(read)

	for {
		// Read messages from the websocket connection and write them to the write
		// channel.
		t, m, err := conn.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				read <- websocketMessage{Err: closeErr}
			} else {
				log.Printf("Could not receive the websocket message: %s", err)
			}
			return
		}

		// Send the message to the channel
		read <- websocketMessage{Type: t, Content: m}
	}
	// If an error happens, close the websocket connection.
}

// Read from a channel (expecting a websocket.send!*** channel) and sends
// any data to the receive channel. Closes the channel if anything goes wrong.
// Shuts down if the value closed is true.
func readChannelLayer(channel string, receive chan asgi.SendCloseAcceptMessage, closed *bool) {
	defer close(receive)
	for !*closed {
		// Read from the channel layer.
		var am asgi.SendCloseAcceptMessage
		c, err := channelLayer.Receive([]string{channel}, true, &am)
		if err != nil {
			log.Printf("could not get the message: %s", err)
			return
		}
		if c != "" {
			// If we read the message from the channel (no timeout) then:
			// Send the received value to the channel.
			receive <- am
		}
	}
}

// Handles an opened websocket connection by forwarding the messages between the
// channel layer and the websocket connection.
func websocketLoop(conn *websocket.Conn, channel string, path string) {
	order := 0
	closeCode := 1006 // Code that is sent to the channel layer. 1006 is used, when no close message was received
	readFromWebsocket := make(chan websocketMessage)
	readFromChannelLayer := make(chan asgi.SendCloseAcceptMessage)
	closed := false

	// In the end: Close the websocket connection and inform the channel layer about it.
	defer func() {
		closed = true
		order++
		channelLayer.Send("websocket.disconnect", &asgi.DisconnectionMessage{
			ReplyChannel: channel,
			Code:         closeCode,
			Path:         path,
			Order:        order,
		})
		conn.Close()
	}()

	// Create goroutines to read to the websocket connection and the channel
	// layer at the same time.
	go readWebsocket(conn, readFromWebsocket)
	go readChannelLayer(channel, readFromChannelLayer, &closed)

	for {
		select {
		case cMessage, ok := <-readFromWebsocket:
			// Received a message from the client
			if !ok {
				// The channel was closed. An error happend. So close the connection
				return
			}

			if cMessage.Err != nil {
				// An error happend while reading from the websocket connection. The
				// usual case is, that the client send a close message (which is handelt
				// as an error). So set the closeCode and exit (and thereby call defer).
				closeCode = cMessage.Err.Code
				return
			}

			// Forward it to the channel layer
			order++
			err := channelLayer.Send("websocket.receive", &asgi.ReceiveMessage{
				ReplyChannel: channel,
				Path:         path,
				Content:      cMessage.Content,
				Type:         cMessage.Type,
				Order:        order,
			})
			if err != nil {
				if asgi.IsChannelFullError(err) {
					// TODO: The specs allow us to retry to sent the message. So do
					conn.CloseHandler()(1013, "Channel layer full.")
				}
				log.Printf("Could not send a message to channel layer: %s", err)
				return
			}

		case r, ok := <-readFromChannelLayer:
			// Received a message from the channel layer
			if !ok {
				// The channel was closed. An error happend. So close the connection
				return
			}

			// Send the message to the websocket connection
			var t int
			var content []byte
			if r.Text != "" {
				t = websocket.TextMessage
				content = []byte(r.Text)
			} else if r.Bytes != nil {
				t = websocket.BinaryMessage
				content = r.Bytes
			} else {
				// Got an message without data. Skip it.
				continue
			}
			if err := conn.WriteMessage(t, content); err != nil {
				log.Printf("Could not send message: %s", err)
				return
			}
		}
	}
}

// Create the reply channel name for a websocket.send channel.
func createWebsocketReplyChannel() (replyChannel string, err error) {
	replyChannel, err = channelLayer.NewChannel("websocket.send!")
	if err != nil {
		return "", asgi.NewForwardError("could not create a new channel name", err)
	}
	return replyChannel, nil
}

// Sends the websocket handshake to the channel layer..
func forwardWebsocketConnection(req *http.Request, channel string) (err error) {
	// Send a connection message to the channel layer.
	err = channelLayer.Send("websocket.connect", &asgi.ConnectionMessage{
		ReplyChannel: channel,
		Scheme:       req.URL.Scheme,
		Path:         req.URL.Path,
		QueryString:  []byte(req.URL.RawQuery),
		Headers:      req.Header,
		Client:       req.Host, //TODO use the right value
		Server:       req.Host,
	})
	if err != nil {
		return asgi.NewForwardError("can not sent message to the channel layer", err)
	}
	return nil
}

// Handles the response after a websocket connection. Returns the websocket connection
// if it was opend.
func receiveAccept(w http.ResponseWriter, req *http.Request, channel string) (*websocket.Conn, error) {
	// Get a message from the channel layer.
	var am asgi.SendCloseAcceptMessage

	// Read from the channel. Try to get a response for httpResponseWait seconds.
	// If there is no response in this time, then break.
	timeout := time.After(httpResponseWait * time.Second)
receiveLoop:
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("did not get a response in time")
		default:
			c, err := channelLayer.Receive([]string{channel}, true, &am)
			if err != nil {
				return nil, asgi.NewForwardError("could not read accept message from the channel layer", err)
			}
			if c != "" {
				// Got a response
				break receiveLoop
			}
		}
	}

	if am.Text != "" || am.Bytes != nil || am.Accept {
		// Finish the websocket handshake by upgrading the http request.
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return nil, asgi.NewForwardError("could not upgrade the http request", err)
		}

		// Send the first data, if there is one.
		if am.Text != "" {
			err = conn.WriteMessage(websocket.TextMessage, []byte(am.Text))
		} else if am.Bytes != nil {
			err = conn.WriteMessage(websocket.BinaryMessage, am.Bytes)
		}
		if err != nil {
			conn.Close()
			if _, ok := err.(*websocket.CloseError); !ok {
				return nil, fmt.Errorf("client closed the connection before first message could be send")
			}
			return nil, asgi.NewForwardError("could not send first message to the websocket connection", err)
		}

		// The connection should be closed again.
		if am.Close != 0 {
			// The connection was opened but should be closed again
			if err = conn.CloseHandler()(am.Close, ""); err != nil {
				return nil, asgi.NewForwardError("Could not close the websocket connection", err)
			}
			// An close message was send to the client but we return the connection
			// anyway, because the connection as to be closed after receiving the
			// close message from the client.
		}
		return conn, nil
	}

	// If we are here, then the websocket connection should not be opened
	if am.Close == 0 {
		// At this point, close has to be set.
		return nil, fmt.Errorf("Got an send/close/accept message with all fields set to nil")
	}
	w.WriteHeader(403)
	return nil, nil
}

// Handels an request that wants to be upgraded to a websocket connection.
// Returns an error if one happen.
func asgiWebsocketHandler(w http.ResponseWriter, req *http.Request) (err error) {
	// Create a reply channel name.
	channel, err := createWebsocketReplyChannel()
	if err != nil {
		return asgi.NewForwardError("can not create new channel for websocket send", err)
	}

	// Send the request to the channel layer and get the reply channel name.
	if err = forwardWebsocketConnection(req, channel); err != nil {
		if asgi.IsChannelFullError(err) {
			w.WriteHeader(503)
			return nil
		}
		return asgi.NewForwardError("could not establish websocket connection", err)
	}

	// Try to receive the answer from the channel layer and open the websocket
	// connection, if it tells us to do.
	conn, err := receiveAccept(w, req, channel)
	if err != nil {
		return fmt.Errorf("could not establish websocket connection: %s", err)
	}
	if conn == nil {
		// The websocket connection was not opend but no error occured. In this
		// case the http response was already send. There is nothing else to do.
		return nil
	}

	// The websocket connection was opened. Handle all messages in a loop
	websocketLoop(conn, channel, req.URL.Path)
	return nil
}
