package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ostcar/goasgiserver/asgi"
)

const httpResponseWait = 10

// Create the reply channel name for a http.response channel.
func createResponseReplyChannel() (replyChannel string, err error) {
	replyChannel, err = channelLayer.NewChannel("http.response!")
	if err != nil {
		return "", asgi.NewForwardError("could not create a new channel name", err)
	}
	return replyChannel, nil
}

// Forwars an http request to the channel layer. Returns the reply channel name.
func forwardHTTPRequest(req *http.Request, channel string) (err error) {
	// Read the body of the request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return asgi.NewForwardError("can not read the body of the request", err)
	}

	// Send the Request message to the channel layer
	err = channelLayer.Send("http.request", &asgi.RequestMessage{
		ReplyChannel: channel,
		HTTPVersion:  req.Proto,
		Method:       req.Method,
		Path:         req.URL.Path,
		Scheme:       req.URL.Scheme,
		QueryString:  []byte(req.URL.RawQuery),
		Headers:      req.Header,
		Body:         body,
		Client:       req.Host, //TODO use the right value
		Server:       req.Host,
	})
	if err != nil {
		return asgi.NewForwardError("can not send message the message to the channel layer", err)
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
			http.Error(w, err.Error(), 503)
		}
		return asgi.NewForwardError("could not send message to the channel layer", err)
	}

	// Receive the response from the channel layer and write it to the http response.
	if err = receiveHTTPResponse(w, channel); err != nil {
		return asgi.NewForwardError("could not receive message from the http response channel", err)
	}
	return nil
}
