package main

import (
	"io"
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
	bigBody := bigReader(1024 * 1024)
	requests := []*http.Request{
		httptest.NewRequest("GET", "http://localhost/", nil),
		httptest.NewRequest("GET", "http://localhost/", strings.NewReader("my body")),
		httptest.NewRequest("GET", "http://localhost:8000/", nil),
		httptest.NewRequest("GET", "https://localhost/", nil),
		httptest.NewRequest("GET", "https://localhost:8430/", nil),
		httptest.NewRequest("GET", "https://localhost", &bigBody),
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
	}
}

func messageIsRequest(m asgi.Message, r *http.Request) (bool, string) {
	if m["method"] != r.Method {
		return false, "message and request have different methods."
	}
	// TODO: compare more values
	// fmt.Println(m)
	return true, ""
}

type dummyReceiver struct {
	message asgi.Message
}

func (r *dummyReceiver) Set(m asgi.Message) error {
	r.message = m
	return nil
}

type bigReader int

func (i *bigReader) Read(p []byte) (int, error) {
	if int(*i) > len(p) {
		*i -= bigReader(len(p))
		return len(p), nil
	}
	v := int(*i)
	*i = 0
	return v, io.EOF
}
