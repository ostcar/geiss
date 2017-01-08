package asgi

import (
	"fmt"
	"math/rand"
	"time"
)

const channelLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// GetChannelnameRandom creates a random string that can be added as suffix to a
// channel name
func GetChannelnameRandom() string {
	var b [12]byte
	for i := range b {
		b[i] = channelLetters[rand.Int63()%int64(len(channelLetters))]
	}
	return string(b[:])
}

// ForwardError is an error that holds another error inside
type ForwardError struct {
	err  error
	text string
}

// NewForwardError creates a new ForwardError.
func NewForwardError(s string, e error) *ForwardError {
	return &ForwardError{
		err:  e,
		text: s,
	}
}

func (e *ForwardError) Error() string {
	return fmt.Sprintf("%s: %s", e.text, e.err)
}
