package redis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ostcar/geiss/asgi"
)

func TestNewChannel(t *testing.T) {
	redisPool = nil
	c := NewChannelLayer(0, "", "test:", 0)

	channel, err := c.NewChannel("myprefix!")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
	if !strings.HasPrefix(channel, "myprefix!") {
		t.Errorf("Expect the channelname to start with \"myprefix!\", got %s", channel)
	}
}

type testMessage struct {
	s string
}

func (t *testMessage) Raw() asgi.Message {
	m := make(asgi.Message)
	m["message"] = t.s
	return m
}

func (t *testMessage) Set(m asgi.Message) error {
	var ok bool
	t.s, ok = m["message"].(string)
	if !ok {
		return fmt.Errorf("Message has wrong format %T", m["message"])
	}
	return nil
}

func TestSendAndReceive(t *testing.T) {
	innerTest := func(block bool) {
		redisPool = nil
		c := NewChannelLayer(0, "", "testsendandreceive:", 0)
		sendMessage := testMessage{
			s: "MyMessage",
		}
		err := c.Send("MyChannel", &sendMessage)
		if err != nil {
			t.Errorf("Did not expect any error, got %s", err)
		}

		var receiveMessage testMessage
		channel, err := c.Receive([]string{"MyChannel"}, block, &receiveMessage)
		if err != nil {
			t.Errorf("Did not expect any error, got %s", err)
		}
		if channel != "MyChannel" {
			t.Errorf("Did expect the channel name MyChannel, got \"%s\"", channel)
		}
		if receiveMessage != sendMessage {
			t.Errorf("Expect the send message to be the received message")
		}
	}
	innerTest(true)
	innerTest(false)
}

func TestSendChannelFull(t *testing.T) {
	redisPool = nil
	c := NewChannelLayer(0, "", "testsendchannelfull:", 1)
	sendMessage := testMessage{
		s: "MyMessage",
	}
	channelname, err := c.NewChannel("TestSendChannelFull")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
	err = c.Send(channelname, &sendMessage)
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
	err = c.Send(channelname, &sendMessage)
	if !asgi.IsChannelFullError(err) {
		t.Errorf("Expected a channel full error, got %s", err)
	}
}
