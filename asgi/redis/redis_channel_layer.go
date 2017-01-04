/*
Package redis implements an asgi channel layer that works with redis.
*/
package redis

import (
	"fmt"

	"goasgiserver/asgi"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/garyburd/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// Sets the time in seconds that Receive() with block=true should wait for a message.
// Currently this can not be more the none second to stay compatible with applications
// that do not send an accept message after an open websocket connection but do
// use reconnecting-websocket.
const blPopTimeout = 10

var luaChanSend *redis.Script

func init() {
	luaChanSend = redis.NewScript(
		2,
		`
		if redis.call('llen', KEYS[2]) >= tonumber(ARGV[3]) then
		    return redis.error_reply("full")
		end
		redis.call('set', KEYS[1], ARGV[1])
		redis.call('expire', KEYS[1], ARGV[2])
		redis.call('rpush', KEYS[2], KEYS[1])
		redis.call('expire', KEYS[2], ARGV[2] + 1)
	`)
}

// ChannelLayer is the main type to use as redis channel layer.
type ChannelLayer struct {
	prefix   string
	expiry   int
	host     string //TODO
	capacity int
}

// NewChannelLayer creates a new RedisChannelLayer
func NewChannelLayer() *ChannelLayer {
	return &ChannelLayer{prefix: "asgi:", expiry: 60, host: ":6379", capacity: 1000}
}

// NewChannel creates a new channelname
func (r *ChannelLayer) NewChannel(channelPrefix string) (channel string, err error) {
	var exists int64
	conn := RedisPool.Get()
	defer conn.Close()

	for {
		// Create a channel name
		channel = channelPrefix + asgi.GetChannelnameRandom()

		// Test if this channel name already exists.
		exists, err = redis.Int64(conn.Do("EXISTS", r.prefix+channel))
		if err != nil {
			err = fmt.Errorf("redis error: %s", err)
			return "", err
		}

		if exists == 0 {
			// If the key does not exist, then we exit this function
			// returning the (free) channelname
			return channel, nil
		}
	}
}

// Send sends a message to a specific channel
func (r *ChannelLayer) Send(channel string, message asgi.SendMessenger) (err error) {
	conn := RedisPool.Get()
	defer conn.Close()

	messageKey := r.prefix + uuid.NewV4().String()
	channelKey := r.prefix + channel

	// Encode the message
	bytes, err := encodeMessage(message)
	if err != nil {
		return err
	}

	// Use the lua script to set both keys
	_, err = luaChanSend.Do(conn, messageKey, channelKey, bytes, r.expiry, r.capacity)
	if err != nil {
		if err.Error() == "full" {
			return asgi.ChannelFullError{}
		}
		return fmt.Errorf("redis luaChanSend error: %s", err)
	}
	return nil
}

func lpopMany(channels []string, conn redis.Conn) (channel, message string, err error) {
	// TODO: use lua
	for _, channel = range channels {
		message, err = redis.String(conn.Do("LPOP", channel))
		if err == redis.ErrNil {
			// The channel is empty
			continue
		} else {
			// An error happend or a value was returned
			break
		}
	}
	return
}

// Receive fills a message from one or more channels
func (r *ChannelLayer) Receive(channels []string, block bool, message asgi.ReceiveMessenger) (channel string, err error) {
	conn := RedisPool.Get()
	defer conn.Close()

	var messageKey string
	var args []interface{}

	if block {
		// If block is True, then this method should wait until there is a message to
		// receive.

		// First, build the arguments for the BLPOP redis command.
		for _, channel := range channels {
			args = append(args, r.prefix+channel)
		}
		args = append(args, blPopTimeout)

		// Call the BLPOP redis command.
		var v []string
		v, err = redis.Strings(conn.Do("BLPOP", args...))
		if err == redis.ErrNil {
			// Got timeout. Nothing to receive
			return "", nil
		} else if err != nil {
			return "", err
		}
		channel = v[0]
		messageKey = v[1]
	} else {
		channel, messageKey, err = lpopMany(channels, conn)
		if err == redis.ErrNil {
			// Nothing to receive
			return "", nil
		} else if err != nil {
			return "", err
		}
	}

	// If we are here, there is a channel with a message in messageKey
	b, err := redis.Bytes(conn.Do("GET", messageKey))
	if err != nil {
		return "", err
	}

	// b is an encoded message. Decode it by creating a message object
	err = decodeMessage(b, message)
	return
}

// decodes a message in msgpack format.
func decodeMessage(bytes []byte, m asgi.ReceiveMessenger) (err error) {
	var raw asgi.Message
	err = msgpack.Unmarshal(bytes, &raw)
	if err != nil {
		err = fmt.Errorf("can not decode message %s, got %s", bytes, err)
		return err
	}
	err = m.Set(raw)
	if err != nil {
		err = fmt.Errorf("can not create a message object from the message %s, got %s", m, err)
		return err
	}
	return nil
}

// encodes a message to the msgpack format.
func encodeMessage(m asgi.SendMessenger) (b []byte, err error) {
	bytes, err := msgpack.Marshal(m.Raw())
	if err != nil {
		err = fmt.Errorf("can not encode message %v, got %s", m, err)
		return nil, err
	}
	return bytes, nil
}
