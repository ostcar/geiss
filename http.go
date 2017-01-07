package main

import (
	"fmt"
	"goasgiserver/asgi"
	"io/ioutil"
	"net/http"
)

// Forwars an http request to the channel layer. Returns the reply channel name.
func forwardHTTPRequest(req *http.Request) (replyChannel string, err error) {
	// Create the reply channel name.
	replyChannel, err = channelLayer.NewChannel("http.response!")
	if err != nil {
		return "", fmt.Errorf("could not create a new channel name: %s", err)
	}

	// Read the body of the request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", fmt.Errorf("can not read the body of the request: %s", err)
	}

	// Create a RequestMessage
	message := asgi.RequestMessage{
		ReplyChannel: replyChannel,
		HTTPVersion:  req.Proto,
		Method:       req.Method,
		Path:         req.URL.Path,
		Scheme:       req.URL.Scheme,
		QueryString:  []byte(req.URL.RawQuery),
		Headers:      req.Header,
		Body:         body,
		Client:       req.Host, //TODO use the right value
		Server:       req.Host,
	}

	// Send the Request message to the channel layer
	if err = channelLayer.Send("http.request", &message); err != nil {
		return "", fmt.Errorf("can not send message %v: %s", message, err)
	}
	return replyChannel, nil
}

// Receives a http response from the channel layer and writes it to the
// http response.
func receiveHTTPResponse(w http.ResponseWriter, replyChannel string) (err error) {
	// Get message from the channel layer.
	var rm asgi.ResponseMessage
	var c string
	for i := 0; i < 100; i++ {
		c, err = channelLayer.Receive([]string{replyChannel}, true, &rm)
		if err != nil {
			return fmt.Errorf("could not get a message: %s", err)
		}
		if c != "" {
			// Got a response
			break
		}
	}
	if c == "" {
		// Did not receive any response. Propably got an timeout.
		return fmt.Errorf("did not get a response in time")
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
	for i:= 0; i < 100 && moreContent; i++ {
		// Get message from the channel layer
		var rcm asgi.ResponseChunkMessage
		c, err := channelLayer.Receive([]string{replyChannel}, true, &rcm)
		if err != nil {
			return fmt.Errorf("could not receive a message: %s", err)
		}
		if c == "" {
			// Did not get any message
			continue
		}

		// Got a message. Reset counter.
		i = 0

		// Write the received content to the http response.
		w.Write(rcm.Content)

		// See if there is still more content.
		moreContent = rcm.MoreContent
	}
	return nil
}

// Handels an http request. Returns an error if one happen.
func asgiHTTPHandler(w http.ResponseWriter, req *http.Request) error {
	// Forward the request to the channel layer and get the reply channel name.
	replyChannel, err := forwardHTTPRequest(req)
	if err != nil {
		return fmt.Errorf("Could not send message to the channel layer: %s", err)
	}

	// Receive the response from the channel layer and write it to the http response.
	if err = receiveHTTPResponse(w, replyChannel); err != nil {
		return fmt.Errorf("Could not receive message from the channel layer: %s", err)
	}
	return nil
}
