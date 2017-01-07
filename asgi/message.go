package asgi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// Message is a dict to send via the channel layer.
// The key has to be a string and the type of the value vary.
type Message map[string]interface{}

// SendMessenger is an type that can be used to send a message through the channel
// layer. Therefore it has the method to convert a structured message to a
// Message dict.
type SendMessenger interface {
	// Converts a Message to a Message dict, that can be send through a asgi channel.
	Raw() Message
}

// ReceiveMessenger is a type that can be used to receive a message through the
// channel layer. Therefore it has the method to convert a Message dict to a
// structured message type.
type ReceiveMessenger interface {
	// Sets the values of the variable with the values of a Message dict.
	Set(Message) error
}

// Messager is an type that can be used to send and receive messages.
type Messager interface {
	SendMessenger
	ReceiveMessenger
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

// RequestMessage is a structured message type, defined by the asgi specs which
// is used to forward an http request from a client to the channel layer.
// This differs from the specs that all fields are uppercase and CamelCase.
// Also the Headers field has the type http.Header and therefore is a dictonary.
// BodyChannel does not default to None but to an empty string.
// Client and Server are strings in the form "host:port". They default to an
// empty string.
type RequestMessage struct {
	ReplyChannel string
	HTTPVersion  string
	Method       string
	Scheme       string
	Path         string
	QueryString  []byte
	RootPath     string
	Headers      http.Header
	Body         []byte
	BodyChannel  string
	Client       string
	Server       string
}

// Raw converts a RequestMessage to a Message dict.
func (r *RequestMessage) Raw() Message {
	m := make(Message)
	m["reply_channel"] = r.ReplyChannel
	m["http_version"] = r.HTTPVersion
	m["method"] = r.Method
	m["scheme"] = r.Scheme
	m["path"] = r.Path
	m["query_string"] = r.QueryString
	m["root_path"] = r.RootPath

	var headers [][2][]byte
	for headerKey, headerValues := range r.Headers {
		for _, headerValue := range headerValues {
			headers = append(headers, [2][]byte{[]byte(strings.ToLower(headerKey)), []byte(headerValue)})
		}
	}
	m["headers"] = headers
	m["body"] = r.Body
	m["body_channel"] = r.BodyChannel
	m["client"], _ = strToHost(r.Client)
	m["server"], _ = strToHost(r.Server)
	return m
}

// TODO: Implement the "Request Body Chunk" message

// ResponseChunkMessage is a structured message type, defined by the asgi specs.
// It is used to forward an response from the channel layer to the client.
// It has to follow a ResponseMessage.
// It differs from the specs that the fields are uppercase and CamelCase.
type ResponseChunkMessage struct {
	Content     []byte
	MoreContent bool
}

// Set fills the values of a ResponseChunkMessage with a the data of a message dict.
func (rm *ResponseChunkMessage) Set(m Message) (err error) {
	var ok bool

	switch t := m["content"].(type) {
	case []byte:
		rm.Content = t
	case nil:
		rm.Content = []byte{}
	default:
		return fmt.Errorf("message has wrong format. \"content\" has to be []byte or nil, not %T", m["content"])
	}

	rm.MoreContent, ok = m["more_content"].(bool)
	if ok == false {
		return fmt.Errorf("message has wrong format. \"more_content\" has to be bool not %T", m["more_content"])
	}
	return nil
}

// ResponseMessage is a structured message type, defined by the asgi specs.
// It is used to forward an response from the channel layer to the client.
// It differs from the specs that the fields are uppercase and CamelCase and that
// Headers is a dictonary and not a list of tuples.
type ResponseMessage struct {
	ResponseChunkMessage
	Status  int
	Headers http.Header
}

// Set fills the values of a ResponseMessage with a the data of a message dict.
func (rm *ResponseMessage) Set(m Message) (err error) {
	var ok bool

	status, ok := m["status"].(uint64)
	if !ok {
		return fmt.Errorf("message has wrong format. \"status\" has to be uint64 not %T", m["status"])
	}
	rm.Status = int(status)

	switch t := m["content"].(type) {
	case []byte:
		rm.Content = t
	case nil:
		rm.Content = []byte{}
	default:
		return fmt.Errorf("message has wrong format. \"content\" has to be []byte or nil, not %T", m["content"])
	}

	rm.MoreContent, ok = m["more_content"].(bool)
	if ok == false {
		return fmt.Errorf("message has wrong format. \"more_content\" has to be bool not %T", m["more_content"])
	}

	rm.Headers = make(http.Header)
	for _, value := range m["headers"].([]interface{}) {
		// value should be a slice of interface{}
		value := value.([]interface{})
		k := string(value[0].([]byte))
		v := string(value[1].([]byte))
		rm.Headers.Add(k, v)
	}
	return nil
}

// TODO: Impelement "Server Push" and "Disconnect"

// ConnectionMessage is a structured message defined by the asgi specs. It is used
// to forware an websocket connection request to the channel layer.
// It differs from the asgi specs that all fields are Uppercase and CamelCase,
// the field Headers is a dict and the fields Client and Server are strings in
// the form "host:port". It has no field order.
type ConnectionMessage struct {
	ReplyChannel string
	Scheme       string
	Path         string
	QueryString  []byte
	RootPath     string
	Headers      http.Header
	Client       string
	Server       string
}

// Raw converts a ConnectionMessage to a Message dict, that can be send through
// the channel layer
func (cm *ConnectionMessage) Raw() Message {
	m := make(Message)
	m["reply_channel"] = cm.ReplyChannel
	m["scheme"] = cm.Scheme
	m["path"] = cm.Path
	m["query_string"] = cm.QueryString
	m["root_path"] = cm.RootPath

	var headers [][2][]byte
	for headerKey, headerValues := range cm.Headers {
		for _, headerValue := range headerValues {
			headers = append(headers, [2][]byte{[]byte(strings.ToLower(headerKey)), []byte(headerValue)})
		}
	}
	m["headers"] = headers
	m["client"], _ = strToHost(cm.Client)
	m["server"], _ = strToHost(cm.Server)
	m["order"] = 0
	return m
}

// ReceiveMessage is message specified by the asgi spec
type ReceiveMessage struct {
	ReplyChannel string
	Path         string
	Content      []byte
	Type         int // See websocket.TextMessage and websocket.BinaryMessage
	Order        int
}

// Raw Converts a ReceiveMessage to a Message
func (cm *ReceiveMessage) Raw() Message {
	m := make(Message)
	m["reply_channel"] = cm.ReplyChannel
	m["path"] = cm.Path
	if cm.Type == websocket.TextMessage {
		m["bytes"] = nil
		m["text"] = string(cm.Content)
	} else if cm.Type == websocket.BinaryMessage {
		m["bytes"] = cm.Content
		m["text"] = nil
	}
	m["order"] = cm.Order
	return m
}

// DisconnectionMessage is a structured message defined by the asgi specs. It is
// send to the channel layer when the connection was closed for any reason.
// It differs from the asgi specs that all fields are Uppercase and CamelCase.
type DisconnectionMessage struct {
	ReplyChannel string
	Code         int
	Path         string
	Order        int
}

// Raw converts a DisconnectionMessage to a Message, that can be send through
// the channel layer.
func (dm *DisconnectionMessage) Raw() Message {
	m := make(Message)
	m["reply_channel"] = dm.ReplyChannel
	m["code"] = dm.Code
	m["path"] = dm.Path
	m["order"] = dm.Order
	return m
}

// SendCloseAcceptMessage is a structured message defined by the asgi specs. It
// is used as answer from the channel layer after a websocket connection and to s
// end data to an open websocket connection.
// It differs from the asgi specs that all fields are Uppercase and CamelCase.
type SendCloseAcceptMessage struct {
	Bytes  []byte
	Text   string
	Close  int
	Accept bool
}

// Set fills the values of a SendCloseAcceptMessage with a the data of a message
// dict.
func (s *SendCloseAcceptMessage) Set(m Message) (err error) {
	switch t := m["bytes"].(type) {
	case []byte:
		s.Bytes = t
	case nil:
		s.Bytes = nil
	default:
		return fmt.Errorf("the field bytes has to be []byte or nil not %T", t)
	}

	switch t := m["text"].(type) {
	case string:
		s.Text = t
	case nil:
		s.Text = ""
	default:
		return fmt.Errorf("the field text has to be string or nil not %T", t)
	}

	if s.Bytes != nil && s.Text != "" {
		return fmt.Errorf("only one of the fields text and bytes can be set at once")
	}

	switch t := m["close"].(type) {
	case bool:
		if t {
			s.Close = 0
		} else {
			s.Close = 1000
		}
	case int:
		s.Close = t
	case nil:
		s.Close = 0
	default:
		return fmt.Errorf("the field \"close\" has to be bool, int or nil, not %T", m["close"])
	}

	switch t := m["accept"].(type) {
	case bool:
		s.Accept = t
	case nil:
		s.Accept = false
	default:
		return fmt.Errorf("the field \"accept\" has to be bool or nil, not %T", m["close"])
	}
	return nil
}
