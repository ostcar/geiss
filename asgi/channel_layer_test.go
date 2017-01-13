package asgi

import (
	"fmt"
	"testing"
)

func TestIsChannelFullError(t *testing.T) {
	err := fmt.Errorf("I am no channel full error")
	if IsChannelFullError(err) {
		t.Errorf("Expected %s not to be a channel full error", err)
	}

	err = ChannelFullError{
		Channel: "test_channel",
	}
	if !IsChannelFullError(err) {
		t.Errorf("Expected %s not to be a channel full error", err)
	}
	if err.Error() != "channel is full: test_channel" {
		t.Errorf("Expected the error string to be \"channel is full: test_channel\" not \"%s\"", err)
	}

	err = NewForwardError("channel full inside", err)
	if !IsChannelFullError(err) {
		t.Errorf("Expected %s not to be a channel full error", err)
	}
}
