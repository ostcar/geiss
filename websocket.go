package main

import (
	"fmt"
	"goasgiserver/asgi"
	"log"
	"net/http"

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
}

// Read from a websocket connection and write any message to the read channel.
func readWebsocket(conn *websocket.Conn, read chan websocketMessage) {
	defer close(read)
	defer conn.Close()
	for {
		// Read messages from the websocket connection and write them to the write
		// channel.
		t, m, err := conn.ReadMessage()
		if err != nil {
			// If the websocket connection was closed, then err will be != nil.
			log.Printf("Could not receive the websocket message: %s", err)
			return
		}

		// Send the message to the channel
		read <- websocketMessage{Type: t, Content: m}
	}
	// If an error happens, close the websocket connection.
}

// Read from a channel (expecting a websocket.send!*** channel) and sends
// any data to the receive channel. Closes the channel if anything goes wrong.
func readChannelLayer(channel string, receive chan asgi.SendCloseAcceptMessage) {
	for {
		// Read from the channel layer.
		var am asgi.SendCloseAcceptMessage
		c, err := channelLayer.Receive([]string{channel}, true, &am)
		if err != nil {
			log.Printf("could not get the message: %s", err)
			break
		}
		if c != "" {
			// If we read the message from the channel (no timeout) then:
			// Send the received value to the channel.
			receive <- am
		}
	}
	close(receive)
}

// Sends the websocket handshake to the channel layer. Returns the name of the
// reply channel.
func forwardWebsocketConnection(req *http.Request) (replyChannel string, err error) {
	// Create the name for the reply channel
	replyChannel, err = channelLayer.NewChannel("websocket.send!")
	if err != nil {
		return "", fmt.Errorf("could not create a new channel name: %s", err)
	}

	// Create the connection message object
	message := asgi.ConnectionMessage{
		ReplyChannel: replyChannel,
		Scheme:       req.URL.Scheme,
		Path:         req.URL.Path,
		QueryString:  []byte(req.URL.RawQuery),
		Headers:      req.Header,
		Client:       req.Host, //TODO use the right value
		Server:       req.Host,
	}

	// Send the message to the channel layer
	if err = channelLayer.Send("websocket.connect", &message); err != nil {
		if asgi.IsChannelFullError(err) {
			// Forward a ChannelFullError
			return "", err
		}
		return "", fmt.Errorf("can not send message %v: %s", message, err)
	}
	return replyChannel, nil
}

// Handles the response after a websocket connection. Returns the websocket connection
// if it was opend.
func receiveAccept(w http.ResponseWriter, req *http.Request, channel string) (*websocket.Conn, *int, error) {
	order := 0

	// Get a message from the channel layer.
	var am asgi.SendCloseAcceptMessage
	n, err := channelLayer.Receive([]string{channel}, true, &am)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get the message: %s", err)
	}

	// TODO: Test the case, that the http request was closed since we got the
	// the answer from the channellayer

	if n == "" {
		// Did not get any message from the channel layer.
		// The specs say, that the handshake should not be finished. But then we are
		// not compatible to reconnecting-websocket that is used by the channels-examples.
		// So for now, we open the websocket connection.
		am.Accept = true
	}

	if am.Text != "" || am.Bytes != nil || am.Accept {
		// Finish the websocket handshake by upgrading the http request.
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("could not upgrade the http request: %s", err)
		}

		// Set a close handler, that informs the channel layer, when the connection
		// is closed.
		conn.SetCloseHandler(func(code int, text string) error {
			order++

			// Send the message to the channel layer. Ignore any error that happens.
			channelLayer.Send("websocket.disconnect", &asgi.DisconnectionMessage{
				ReplyChannel: channel,
				Code:         code,
				Path:         req.URL.Path,
				Order:        order,
			})

			// Call the original handler
			conn.SetCloseHandler(nil)
			return conn.CloseHandler()(code, text)
		})

		// Send the first data, if there is one.
		if am.Text != "" {
			conn.WriteMessage(websocket.TextMessage, []byte(am.Text))
		} else if am.Bytes != nil {
			conn.WriteMessage(websocket.BinaryMessage, am.Bytes)
		}

		// The connection should be closed again.
		if am.Close != 0 {
			// The connection was opened but should be closed again
			if conn.CloseHandler()(am.Close, ""); err != nil {
				return nil, nil, fmt.Errorf("Could not close the websocket connection: %s", err)
			}
		}
		return conn, &order, nil
	}

	// If we are here, then the websocket connection should not be opend
	if am.Close == 0 {
		// At this point, close has to be set.
		return nil, nil, fmt.Errorf("Got an send/close/accept message with all fields set to nil")
	}
	w.WriteHeader(403)
	return nil, nil, nil
}

// Handels an request that wants to be upgraded to a websocket connection.
// Returns an error if one happen.
func asgiWebsocketHandler(w http.ResponseWriter, req *http.Request) error {
	// Send the request to the channel layer and get the reply channel name.
	replyChannel, err := forwardWebsocketConnection(req)
	if err != nil {
		if asgi.IsChannelFullError(err) {
			w.WriteHeader(503)
			return nil
		}
		return fmt.Errorf("could not establish websocket connection: %s", err)
	}

	// Try to receive the answer from the channel layer and open the websocket
	// connection, if it tells us to do.
	conn, order, err := receiveAccept(w, req, replyChannel)
	if err != nil {
		return fmt.Errorf("could not establish websocket connection: %s", err)
	}
	if conn == nil {
		// The websocket connection was not opend but no error occured. In this
		// case the http response was already send.
		return nil
	}

	// Close the websocket connection in the end.
	defer conn.Close()

	// Create goroutines to read to the websocket connection and the channel
	// layer at the same time.
	read := make(chan websocketMessage)
	receive := make(chan asgi.SendCloseAcceptMessage)
	go readWebsocket(conn, read)
	go readChannelLayer(replyChannel, receive)

	for {
		select {
		case r, ok := <-read:
			// Received a message from the client
			if !ok {
				// The channel was closed. An error happend. So close the connection
				return nil
			}
			*order++
			message := asgi.ReceiveMessage{
				ReplyChannel: replyChannel,
				Path:         req.URL.Path,
				Content:      r.Content,
				Type:         r.Type,
				Order:        *order,
			}

			// Forward it to the channel layer
			if err = channelLayer.Send("websocket.receive", &message); err != nil {
				if asgi.IsChannelFullError(err) {
					conn.CloseHandler()(websocket.CloseTryAgainLater, "Channel layer full.")
				}
				log.Printf("Could not send a message to channel layer: %s", err)
				return nil
			}

		case r, ok := <-receive:
			// Received a message from the channel layer
			if !ok {
				// The channel was closed. An error happend. So close the connection
				return nil
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
			if err = conn.WriteMessage(t, content); err != nil {
				log.Printf("Could not send message: %s", err)
				return nil
			}
		}
	}
}
