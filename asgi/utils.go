package asgi

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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

// strToHost converts a string in the form "host:port" to an two element array
// where the first element is the host as string and the second argument is
// the port as integer.
func strToHost(host string) (hp [2]interface{}, err error) {
	s := strings.Split(host, ":")
	if len(s) != 2 {
		err = fmt.Errorf("host has wrong format: %s", host)
		return
	}
	hp[0] = s[0]
	hp[1], err = strconv.Atoi(s[1])
	if err != nil {
		err = fmt.Errorf("can not convert %s to int", s[1])
	}
	return
}

// Converts http.Headers in the form that the asgi specs
// expects them
func convertHeader(httpHeaders http.Header) (headers [][2][]byte) {
	for headerKey, headerValues := range httpHeaders {
		for _, headerValue := range headerValues {
			headers = append(headers, [2][]byte{[]byte(strings.ToLower(headerKey)), []byte(headerValue)})
		}
	}
	return
}
