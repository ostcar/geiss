package asgi

import (
	"fmt"
	"math/rand"
	"net"
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
func strToHost(hostport string) (hp [2]interface{}, err error) {
	if hostport == "" {
		err = fmt.Errorf("The argument for strToHost can not be empty")
		return
	}
	host, port, err := net.SplitHostPort(hostport)
	if err != nil && strings.Contains(err.Error(), "missing port in address") {
		// Try once more with the port 80
		err = nil
		host, port, err = net.SplitHostPort(hostport + ":80")
	}
	if err != nil {
		return
	}
	hp[0] = host
	hp[1], err = strconv.Atoi(port)
	if err != nil {
		err = fmt.Errorf("can not convert %s to int", port)
		return
	}
	return
}

// ConvertHeader converts http.Headers in the form that the asgi specs
// expects them
func ConvertHeader(httpHeaders http.Header) (headers [][2][]byte) {
	for headerKey, headerValues := range httpHeaders {
		for _, headerValue := range headerValues {
			headers = append(headers, [2][]byte{[]byte(strings.ToLower(headerKey)), []byte(headerValue)})
		}
	}
	return
}
