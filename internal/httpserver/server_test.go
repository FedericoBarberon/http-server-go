package httpserver

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestServerResponse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })

	sv := NewServer()
	go sv.Serve(l)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatal(err)
	}

	resp, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}

	expectedPrefix := []byte("HTTP/1.1 200")
	if !bytes.HasPrefix(resp, expectedPrefix) {
		t.Errorf("got %s", resp)
	}
}
