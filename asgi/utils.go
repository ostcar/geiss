package asgi

import (
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
