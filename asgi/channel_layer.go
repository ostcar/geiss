/*
Package asgi implements the asgi specs defined at

https://channels.readthedocs.io/en/latest/asgi.html

On the networklayer it should be fully compatible with the spec. But the methods
and functions defined differ from the spec to be more go-like.
*/
package asgi

import "fmt"

// ChannelLayer is a interface with the base methods to send and receive messages
type ChannelLayer interface {

	// Send sends a message to a channel. The first argument has to be a channel
	// name. The second argument has to be a raw message dict.
	Send(channel string, message Message) (err error)

	// Receive listens to a list of channels and gets a message from it. The first
	// argument is a slice (list) of channel names. The second argument determinis
	// if the method should block until there is a message to receive or if the
	// method should return at once.
	// The method returns the channel name from which a value was received or an
	// empty string, if no value could be received. The second return value is the
	// received raw message.
	Receive(channels []string, block bool) (channelname string, message Message, err error)

	// NewChannel taks a prefix of a channlname and adds a unique suffix. It also
	// returns an error, if some happen.
	NewChannel(string) (string, error)
}

// ChannelFullError is used, when a channel is full
type ChannelFullError struct {
	Channel string
}

func (e ChannelFullError) Error() string {
	return fmt.Sprintf("channel is full: %s", e.Channel)
}

// IsChannelFullError returns true, if the error is an ChannelFullError
// If the error is a ForwardError, then return true, if one of the inner errors
// is a ChannelFullError
func IsChannelFullError(err error) bool {
	switch t := err.(type) {
	case ChannelFullError:
		return true
	case *ForwardError:
		return IsChannelFullError(t.err)
	}
	return false
}
