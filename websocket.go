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

// Processes the read and write actions of a websocket connection. You can read
// from the read channel to get messages and write them to the write channel
// to send them. The websocket connection is closed, if the write channels is
// closed.
func processWebsocket(conn *websocket.Conn, read, write chan websocketMessage) {
	go func() {
		for message := range write {
			// Write messages from the write channel to the websocket connection.
			err := conn.WriteMessage(message.Type, message.Content)
			if err != nil {
				log.Printf("Could not send message: %s", err)
				break
			}
		}
		// If an error happens or the write channel is closed: close the websocket
		// connection.
		conn.Close()
	}()

	for {
		// Read messages from the websocket connection and write them to the write
		// channel.
		t, m, err := conn.ReadMessage()
		if err != nil {
			// If the websocket connection was closed, then err will be != nil.
			log.Printf("Could not receive the websocket message: %s", err)
			break
		}
		// Send the message to the channel
		read <- websocketMessage{Type: t, Content: m}
	}
	// If an error happens, close the websocket connection.
	conn.Close()
}

// Read from a channel (expecting a websocket.send!*** channel) and sends
// any data to the receive channel. Closes the channel if anything goes wrong.
func processChannelLayerReceive(channel string, receive chan asgi.SendCloseAcceptMessage) {
	for {
		// Read from the channel layer.
		var am asgi.SendCloseAcceptMessage
		_, err := channelLayer.Receive([]string{channel}, true, &am)
		if err != nil {
			log.Printf("could not get the message: %s", err)
			break
		}

		// Send the received value to the channel.
		receive <- am
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
func receiveAccept(w http.ResponseWriter, req *http.Request, channel string) (conn *websocket.Conn, err error) {
	// Get a message from the channel layer.
	var am asgi.SendCloseAcceptMessage
	n, err := channelLayer.Receive([]string{channel}, true, &am)
	if err != nil {
		return nil, fmt.Errorf("could not get the message: %s", err)
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
		conn, err = upgrader.Upgrade(w, req, nil)
		if err != nil {
			return nil, fmt.Errorf("could not upgrade the http request: %s", err)
		}

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
				return nil, fmt.Errorf("Could not close the websocket connection: %s", err)
			}
		}
		return conn, nil
	}

	// If we are here, then the websocket connection should not be opend
	if am.Close == 0 {
		// At this point, close has to be set.
		return nil, fmt.Errorf("Got an send/close/accept message with all fields set to nil")
	}
	w.WriteHeader(403)
	return nil, nil
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
	conn, err := receiveAccept(w, req, replyChannel)
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

	// Create goroutines to read/write to the websocket connection and the channel
	// layer at the same time.
	read := make(chan websocketMessage)
	write := make(chan websocketMessage)
	receive := make(chan asgi.SendCloseAcceptMessage)
	order := 0
	go processWebsocket(conn, read, write)
	go processChannelLayerReceive(replyChannel, receive)

	for {
		select {
		case r := <-read:
			// Received a message from the client
			order++
			message := asgi.ReceiveMessage{
				ReplyChannel: replyChannel,
				Path:         req.URL.Path,
				Content:      r.Content,
				Type:         r.Type,
				Order:        order,
			}

			// Forward it to the channel layer
			err = channelLayer.Send("websocket.receive", &message)
			if err != nil {
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
			if r.Text != "" {
				write <- websocketMessage{
					Type:    websocket.TextMessage,
					Content: []byte(r.Text),
				}
			} else {
				write <- websocketMessage{
					Type:    websocket.BinaryMessage,
					Content: r.Bytes,
				}
			}
		}
	}
}