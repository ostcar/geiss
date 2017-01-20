package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ostcar/geiss/asgi"
)

const httpResponseWait = 30

// Create the reply channel name for a http.response channel.
func createResponseReplyChannel() (replyChannel string, err error) {
	replyChannel, err = channelLayer.NewChannel("http.response!")
	if err != nil {
		return "", asgi.NewForwardError("could not create a new channel name", err)
	}
	return replyChannel, nil
}

// Forwars an http request to the channel layer. Returns the reply channel name.
func forwardHTTPRequest(req *http.Request, replyChannel string) (err error) {
	const readSize = 500 * 1024 // Read 500kb at once
	var eob bool                // end of body
	var bodyChannel string
	bodyBuf := make([]byte, readSize)

	// Read the firt part of the body
	n, err := req.Body.Read(bodyBuf)
	if err != nil && err != io.EOF {
		return asgi.NewForwardError("can not read the body of the request", err)
	}
	eob = err == io.EOF

	// If there is a second part of the body, then create a channel to read from it.
	if !eob {
		bodyChannel, err = channelLayer.NewChannel("http.request.body?")
		if err != nil {
			return asgi.NewForwardError("can not create new channel name", err)
		}
	}

	host := req.Host
	if req.TLS != nil && !strings.Contains(req.Host, ":") {
		// If no port was set in the host explicitly, the asgi implementation uses
		// 80 as default. So if the request is a https request, we have to manualy
		// set it to 443
		host = req.Host + ":443"
	}

	// Send the Request message to the channel layer
	err = channelLayer.Send("http.request", &asgi.RequestMessage{
		ReplyChannel: replyChannel,
		HTTPVersion:  req.Proto,
		Method:       req.Method,
		Path:         req.URL.Path,
		Scheme:       req.URL.Scheme,
		QueryString:  []byte(req.URL.RawQuery),
		Headers:      req.Header,
		Body:         bodyBuf[:n],
		BodyChannel:  bodyChannel,
		Client:       req.RemoteAddr,
		Server:       host,
	})
	if err != nil {
		// If err is an channel full error, we forward it. The asgi specs define, that
		// we should not retry in this case, but return a 503.
		return asgi.NewForwardError("can not send the message to the channel layer", err)
	}

	for !eob {
		// Read more content from the body
		n, err := req.Body.Read(bodyBuf)
		if err != nil && err != io.EOF {
			return asgi.NewForwardError("can not read the body of the request", err)
		}
		eob = err == io.EOF

		for i := 0; i < 1000; i++ {
			err = channelLayer.Send(bodyChannel, &asgi.RequestBodyChunkMessage{
				Content:     bodyBuf[:n],
				Closed:      false, // TODO test if the connection is closed
				MoreContent: !eob,
			})
			if err != nil {
				if asgi.IsChannelFullError(err) {
					// If the channel is full, then try again.
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return asgi.NewForwardError("can not send the message to the channel layer", err)
			}
			break
		}
	}
	return nil
}

// Receives a http response from the channel layer and writes it to the http response.
func receiveHTTPResponse(w http.ResponseWriter, channel string) (err error) {
	// Get message from the channel layer.
	var rm asgi.ResponseMessage
	var c string

	// Read from the channel. Try to get a response for httpResponseWait seconds.
	// If there is no response in this time, then break.
	timeout := time.After(httpResponseWait * time.Second)
responseLoop:
	for {
		select {
		case <-timeout:
			return fmt.Errorf("did not get a response in time")
		default:
			c, err = channelLayer.Receive([]string{channel}, true, &rm)
			if err != nil {
				if asgi.IsChannelFullError(err) {
					// If the channel is full, then we try again.
					continue responseLoop
				}
				return asgi.NewForwardError("can not get a receive message from the channel laser", err)
			}
			if c != "" {
				// Got a response
				break responseLoop
			}
		}
	}

	// Write the headers from the response message to the http resonse
	for k, v := range rm.Headers {
		w.Header()[k] = v
	}

	// Set the status code of the http response and write the first part of the content
	w.WriteHeader(rm.Status)
	w.Write(rm.Content)

	// If there is more content, then receive it
	moreContent := rm.MoreContent
	timeout = time.After(httpResponseWait * time.Second)
responseChunkLoop:
	for moreContent {
		select {
		case <-timeout:
			// We got the information, that more content is comming, but it don't.
			// So just return and thereby close the http connection.
			return nil
		default:
			// Get message from the channel layer
			var rcm asgi.ResponseChunkMessage
			c, err := channelLayer.Receive([]string{channel}, true, &rcm)
			if err != nil {
				if asgi.IsChannelFullError(err) {
					// If the channel is full, then we try again.
					continue responseChunkLoop
				}
				return asgi.NewForwardError("can not get a receive message from the channel laser", err)
			}
			if c == "" {
				// Did not get any message
				continue responseChunkLoop
			}

			// Write the received content to the http response.
			w.Write(rcm.Content)

			// See if there is still more content.
			moreContent = rcm.MoreContent
		}
	}
	return nil
}

// Handels an http request. Returns an error if it happens.
func asgiHTTPHandler(w http.ResponseWriter, req *http.Request) error {
	// Get the reply channel name
	channel, err := createResponseReplyChannel()
	if err != nil {
		return asgi.NewForwardError("can not create new channel for http respons", err)
	}

	// Forward the request to the channel layer and get the reply channel name.
	if err = forwardHTTPRequest(req, channel); err != nil {
		if asgi.IsChannelFullError(err) {
			handleError(w, err.Error(), 503)
			return nil
		}
		return asgi.NewForwardError("could not send message to the channel layer", err)
	}

	// Receive the response from the channel layer and write it to the http response.
	if err = receiveHTTPResponse(w, channel); err != nil {
		return asgi.NewForwardError("could not receive message from the http response channel", err)
	}
	return nil
}
