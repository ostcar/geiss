package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostcar/geiss/asgi"
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

		var d dummyMessanger
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
			t.Errorf(
				"Expected the reply channel in the message to be \"some-channel\". message: %v",
				d.message,
			)
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

	var d1 dummyMessanger
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

	var d2 dummyMessanger
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

func TestReceiveHTTPResponse(t *testing.T) {
	var d1 dummyMessanger
	d1.message = make(asgi.Message)

	headers := [][2][]byte{[2][]byte{[]byte("MyKey"), []byte("MyValue")}}
	d1.message["status"] = 200
	d1.message["content"] = []byte("foobar")
	d1.message["more_content"] = false
	d1.message["headers"] = headers

	for _, d := range []dummyMessanger{d1} {
		err := channelLayer.Send("my_response_channel", &d)
		if err != nil {
			t.Errorf("Did not expect an error, got: %s", err)
		}
		response := httptest.NewRecorder()
		err = receiveHTTPResponse(response, "my_response_channel")
		if err != nil {
			t.Fatalf("Did not expect an error, got: %s", err)
		}
		if compareMessageHeader(asgi.ConvertHeader(response.Header()), headers) {
			t.Errorf("Received wrong headers. got %v", response.Header())
		}
		if response.Code != d.message["status"].(int) {
			t.Errorf(
				"Reveived wrong status code. Expected %d, got %d",
				d.message["status"].(int),
				response.Code,
			)
		}
		if !bytes.Equal(response.Body.Bytes(), d.message["content"].([]byte)) {
			t.Errorf(
				"Reveived wrong content. Expected %s, got %s",
				d.message["content"].([]byte),
				response.Body.Bytes(),
			)
		}
	}
}

func TestReceiveBigHTTPResponse(t *testing.T) {
	var d1a, d1b dummyMessanger
	d1a.message = make(asgi.Message)
	d1b.message = make(asgi.Message)
	headers := [][2][]byte{[2][]byte{[]byte("OtherKey"), []byte("OtherValue")}}
	d1a.message["status"] = 201
	d1a.message["content"] = []byte("foobar")
	d1a.message["more_content"] = true
	d1a.message["headers"] = headers
	err := channelLayer.Send("my_response_channel", &d1a)
	if err != nil {
		t.Errorf("Did not expect an error, got: %s", err)
	}

	d1b.message["content"] = []byte("even more content")
	d1b.message["more_content"] = false
	err = channelLayer.Send("my_response_channel", &d1b)
	if err != nil {
		t.Errorf("Did not expect an error, got: %s", err)
	}

	response := httptest.NewRecorder()
	err = receiveHTTPResponse(response, "my_response_channel")
	if err != nil {
		t.Fatalf("Did not expect an error, got: %s", err)
	}
	if compareMessageHeader(asgi.ConvertHeader(response.Header()), headers) {
		t.Errorf("Received wrong headers. got %v", response.Header())
	}
	if response.Code != d1a.message["status"].(int) {
		t.Errorf(
			"Reveived wrong status code. Expected %d, got %d",
			d1a.message["status"].(int),
			response.Code,
		)
	}
	fullContent := d1a.message["content"].([]byte)
	fullContent = append(fullContent, d1b.message["content"].([]byte)...)
	if !bytes.Equal(response.Body.Bytes(), fullContent) {
		t.Errorf("Reveived wrong content. Expected %s, got %s", fullContent, response.Body.Bytes())
	}
}

func TestAsgiHTTPHandler(t *testing.T) {
	go func() {
		// Test asgi application server, that reads from http.request and returns
		// the content.
		var request, response dummyMessanger
		_, err := channelLayer.Receive([]string{"http.request"}, true, &request)
		if err != nil {
			t.Errorf("Did not expect an error, got : %s", err)
		}

		response.message = make(asgi.Message)
		headers := [][2][]byte{[2][]byte{[]byte("OtherKey"), []byte("OtherValue")}}
		replyChannel := request.message["reply_channel"].(string)
		response.message["status"] = 200
		response.message["content"] = request.message["body"].([]byte)
		response.message["more_content"] = false
		response.message["headers"] = headers
		err = channelLayer.Send(replyChannel, &response)
		if err != nil {
			t.Errorf("Did not expect an error, got: %s", err)
		}
	}()

	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", strings.NewReader("ping"))
	err := asgiHTTPHandler(response, request)
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
}
