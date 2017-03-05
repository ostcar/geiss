package main

// This file contains functions and types, that make testing easier

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ostcar/geiss/asgi"
)

func messageIsRequest(m asgi.Message, r *http.Request) (bool, string) {
	if m["method"] != r.Method {
		return false, "message and request have a different http method."
	}
	messageBody := m["body"].([]byte)
	requestBody, ok := r.Body.(*testBody)
	if !ok {
		if len(messageBody) > 0 {
			return false, fmt.Sprintf("got an body in the message, but non in the requets: %s", messageBody)
		}
	} else {
		if !bytes.Equal([]byte(requestBody.backup), messageBody) {
			return false, "message and request have different body."
		}
	}
	return true, ""
}

type dummyMessanger struct {
	message asgi.Message
}

func (r *dummyMessanger) Set(m asgi.Message) error {
	r.message = m
	return nil
}

func (r *dummyMessanger) Raw() asgi.Message {
	return r.message
}

type bigReader int

func (i *bigReader) Read(p []byte) (int, error) {
	if int(*i) > len(p) {
		*i -= bigReader(len(p))
		return len(p), nil
	}
	v := int(*i)
	*i = 0
	return v, io.EOF
}

type testBody struct {
	body   io.Reader
	backup string
	step   int
}

func newTestBody(s string) *testBody {
	return &testBody{
		body:   strings.NewReader(s),
		backup: s,
	}
}

func newSlowTestBody(s string, step int) *testBody {
	return &testBody{
		body:   strings.NewReader(s),
		backup: s,
		step:   step,
	}
}

func (t *testBody) Read(p []byte) (n int, err error) {
	if t.step > 0 {
		p = p[:t.step]
	}
	return t.body.Read(p)
}

func (t *testBody) Close() error {
	return nil
}

func compareMessageHeader(header1, header2 [][2][]byte) bool {
	if len(header1) != len(header2) {
		return false
	}

	for i := range header1 {
		if !bytes.Equal(header1[i][0], header1[i][1]) {
			return false
		}
	}
	return true
}
