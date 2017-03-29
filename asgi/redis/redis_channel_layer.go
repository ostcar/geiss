/*
Package redis implements an asgi channel layer that works with redis.
*/
package redis

import (
	"fmt"
	"log"
	"strings"

	"github.com/ostcar/geiss/asgi"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/garyburd/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// Sets the time in seconds that Receive() with block=true should wait for a message.
const blPopTimeout = 3

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
	host     string
	capacity int
}

// NewChannelLayer creates a new RedisChannelLayer
func NewChannelLayer(expiry int, host string, prefix string, capacity int) *ChannelLayer {
	// TODO: Work with more then one host
	if expiry == 0 {
		expiry = 60
	}
	if host == "" {
		host = ":6379"
	}
	if prefix == "" {
		prefix = "asgi:"
	}
	if capacity == 0 {
		capacity = 100
	}
	if redisPool != nil {
		log.Fatalln("Redis pool already set. Can not create a second one.")
	}
	CreateRedisPool(host)
	return &ChannelLayer{prefix: prefix, expiry: expiry, host: host, capacity: capacity}
}

// NewChannel creates a new channelname
func (r *ChannelLayer) NewChannel(channelPrefix string) (channel string, err error) {
	var exists int64
	conn := redisPool.Get()
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
func (r *ChannelLayer) Send(channel string, message asgi.Message) (err error) {
	conn := redisPool.Get()
	defer conn.Close()

	messageKey := r.prefix + uuid.NewV4().String()
	channelKey := r.prefix + channel

	// Encodes a message to the msgpack format.
	bytes, err := msgpack.Marshal(message)
	if err != nil {
		return fmt.Errorf("can not encode message %v, got %s", message, err)
	}

	// Use the lua script to set both keys
	_, err = luaChanSend.Do(conn, messageKey, channelKey, bytes, r.expiry, r.capacity)
	if err != nil {
		if err.Error() == "full" {
			return asgi.ChannelFullError{
				Channel: channel,
			}
		}
		return fmt.Errorf("redis luaChanSend error: %s", err)
	}
	return nil
}

func lpopMany(
	prefix string,
	channels []string,
	conn redis.Conn,
) (channel, message string, err error) {
	// TODO: use lua
	for _, channel = range channels {
		message, err = redis.String(conn.Do("LPOP", prefix+channel))
		if err == redis.ErrNil {
			// The channel is empty
			continue
		} else {
			// An error happened or a value was returned
			return channel, message, err
		}
	}
	// Did not receive anything
	return "", "", redis.ErrNil
}

// Receive reads from a channel and returns a raw message objekt
func (r *ChannelLayer) Receive(
	channels []string,
	block bool) (channel string, message asgi.Message, err error) {

	conn := redisPool.Get()
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
			return "", nil, nil
		} else if err != nil {
			return "", nil, err
		}
		channel = strings.TrimPrefix(v[0], r.prefix)
		messageKey = v[1]
	} else {
		channel, messageKey, err = lpopMany(r.prefix, channels, conn)
		if err == redis.ErrNil {
			// Nothing to receive
			return "", nil, nil
		} else if err != nil {
			return "", nil, err
		}
	}

	// If we are here, there is a channel with a message in messageKey
	b, err := redis.Bytes(conn.Do("GET", messageKey))
	if err != nil {
		return "", nil, err
	}

	// b is an encoded message. First decode it.
	if err = msgpack.Unmarshal(b, &message); err != nil {
		return "", nil, fmt.Errorf("can not decode message %s, got %s", b, err)
	}

	// check if there is a channel information in the raw message
	if v, ok := message["__asgi_channel__"]; ok {
		channel = v.(string)
		delete(message, "__asgi_channel__")
	}
	return
}
