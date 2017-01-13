package asgi

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

func TestGetChannelnameRandom(t *testing.T) {
	name1 := GetChannelnameRandom()
	name2 := GetChannelnameRandom()
	if name1 == name2 {
		t.Error("Called GetChannelnameRandom two times. Got two identical strings.")
	}
}

func TestForwardError(t *testing.T) {
	err := fmt.Errorf("I am an error")
	err = NewForwardError("Error catched", err)
	if err.Error() != "Error catched: I am an error" {
		t.Errorf("Wrong error message: %s", err.Error())
	}
}

func TestStrToHost(t *testing.T) {
	hp, err := strToHost("localhost:12345")
	if err != nil {
		t.Errorf("Did not expect any error, got `%s`", err)
	}
	if hp != [2]interface{}{"localhost", 12345} {
		t.Errorf("Expected the two values localhost and 12345, got %s", hp)
	}

	for _, wrongStr := range []string{"local123", "local:123:45", "local:one"} {
		_, err = strToHost(wrongStr)
		if err == nil {
			t.Errorf("Expected an error. Got non")
		}
	}
}

func TestConvertHeader(t *testing.T) {
	httpHeader := make(http.Header)
	httpHeader.Set("foo", "value")
	httpHeader.Add("Bar", "value1")
	httpHeader.Add("bar", "value2")

	headers := convertHeader(httpHeader)

	if len(headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(headers))
	}

	h1 := [2][]byte{[]byte("foo"), []byte("value")}
	h2 := [2][]byte{[]byte("bar"), []byte("value1")}
	h3 := [2][]byte{[]byte("bar"), []byte("value2")}
	expects := [][2][]byte{h1, h2, h3}
	for _, expect := range expects {
		if !headerInSlice(expect, headers) {
			t.Errorf("Could not find header %s in %s", expect, headers)
		}
	}
}

func headerInSlice(header [2][]byte, headers [][2][]byte) bool {
	for i, h := range headers {
		if bytes.Equal(headers[i][0], h[0]) && bytes.Equal(headers[i][1], h[1]) {
			return true
		}
	}
	return false
}
