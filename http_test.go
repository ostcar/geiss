package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostcar/geiss/asgi/redis"
)

func init() {
	channelLayer = redis.NewChannelLayer(0, "", "http_test:", 0)
}

func TestCreateResponseReplyChannel(t *testing.T) {
	channel1, err := createResponseReplyChannel()
	if err != nil {
		t.Errorf("Did not expect an error, got : %s", err)
	}
	if !strings.HasPrefix(channel1, "http.response!") {
		t.Errorf("Expected the channel name to have a prefix, got: %s", channel1)
	}

	channel2, err := createResponseReplyChannel()
	if err != nil {
		t.Errorf("Did not expect an error, got : %s", err)
	}
	if channel1 == channel2 {
		t.Errorf("Got two times the same channel name: %s and: %s", channel1, channel2)
	}
}

func TestForwardHTTPRequest(t *testing.T) {
	bigBody := bigReader(500 * 1024)
	requests := []*http.Request{
		httptest.NewRequest("GET", "http://localhost/", nil),
		httptest.NewRequest("GET", "http://localhost/", newTestBody(strings.NewReader("my body"))),
		httptest.NewRequest("GET", "http://localhost:8000/", nil),
		httptest.NewRequest("GET", "https://localhost/", nil),
		httptest.NewRequest("GET", "https://localhost:8430/", nil),
		httptest.NewRequest("GET", "https://localhost", newTestBody(&bigBody)),
	}

	for _, request := range requests {
		// Send a request to the channel layer
		err := forwardHTTPRequest(request, "some-channel")
		if err != nil {
			t.Errorf("Did not expect an error, got : %s", err)
		}

		var d dummyReceiver
		channel, err := channelLayer.Receive([]string{"http.request"}, false, &d)
		if err != nil {
			t.Errorf("Did not expect an error, got : %s", err)
		}
		if channel == "" {
			t.Errorf("Expected a message on the http.request channel")
		}
		if ok, errMessage := messageIsRequest(d.message, request); !ok {
			t.Errorf("Expcted the message in the channellayer to be the request: %s", errMessage)
		}
		if d.message["reply_channel"] != "some-channel" {
			t.Errorf("Expected the reply channel in the message to be \"some-channel\". message: %v", d.message)
		}
		if d.message["body_channel"] != "" {
			t.Errorf("Expected the body_channel to be empty")
		}
	}
}

func TestForwardBigHTTPRequest(t *testing.T) {
	bigBody := bigReader(1000 * 1024)
	request := httptest.NewRequest("GET", "https://localhost", newTestBody(&bigBody))
	err := forwardHTTPRequest(request, "some-channel")
	if err != nil {
		t.Errorf("Did not expect an error, got : %s", err)
	}
	var d1 dummyReceiver
	channel, err := channelLayer.Receive([]string{"http.request"}, false, &d1)
	if err != nil {
		t.Errorf("Did not expect an error, got : %s", err)
	}
	if channel == "" {
		t.Errorf("Expected a message on the http.request channel")
	}
	bodyChannel := d1.message["body_channel"].(string)
	if bodyChannel == "" {
		t.Errorf("Expected more content")
	}

	var d2 dummyReceiver
	channel, err = channelLayer.Receive([]string{bodyChannel}, false, &d2)
	if err != nil {
		t.Errorf("Did not expect an error, got : %s", err)
	}
	if channel == "" {
		t.Error("Expected a message on the http.request channel")
	}
	if d2.message["more_content"].(bool) {
		t.Error("Expected no more content.")
	}
	fullBody := make([]byte, 0)
	fullBody = append(fullBody, d1.message["body"].([]byte)...)
	fullBody = append(fullBody, d2.message["content"].([]byte)...)
	if !bytes.Equal(fullBody, request.Body.(*testBody).backup.Bytes()) {
		t.Error("Expected the body of both messages to be the same as the request body.")
	}
}
