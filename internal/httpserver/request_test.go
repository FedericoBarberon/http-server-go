package httpserver_test

import (
	"bytes"
	"fmt"
	"http-server/internal/httpserver"
	"testing"
)

func TestNewRequest(t *testing.T) {
	t.Run("creates a valid GET request with no body", func(t *testing.T) {
		request, err := httpserver.NewRequest(httpserver.MethodGet, "/files/abc.txt", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if request.Method != httpserver.MethodGet {
			t.Errorf("method = %q, want %q", request.Method, httpserver.MethodGet)
		}
		if request.Path != "/files/abc.txt" {
			t.Errorf("path = %q, want %q", request.Path, "/files/abc.txt")
		}
		if _, ok := request.Headers["content-length"]; ok {
			t.Errorf("headers = %#v, expected no content-length for an empty body", request.Headers)
		}
	})

	t.Run("normalizes header keys to lowercase", func(t *testing.T) {
		request, err := httpserver.NewRequest(httpserver.MethodGet, "/",
			map[string]string{"Host": "example.com"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if request.Headers["host"] != "example.com" {
			t.Errorf("host header = %q, want %q", request.Headers["host"], "example.com")
		}
		if _, ok := request.Headers["Host"]; ok {
			t.Errorf("headers = %#v, expected no uppercase key to remain", request.Headers)
		}
	})

	// Asunción: colisión después de normalizar (dos keys que solo difieren en
	// case) se rechaza en vez de dejar que gane una en silencio. El orden de
	// iteración del map no es determinístico, así que el test solo puede
	// afirmar que hay error, no cuál valor "ganó".
	t.Run("rejects headers that collide after normalization", func(t *testing.T) {
		_, err := httpserver.NewRequest(httpserver.MethodGet, "/",
			map[string]string{"Host": "example.com", "host": "other.com"}, nil)
		if err == nil {
			t.Fatal("expected an error for headers colliding after normalization")
		}
	})

	t.Run("normalizes header keys before validating Content-Length", func(t *testing.T) {
		request, err := httpserver.NewRequest(httpserver.MethodPost, "/files/new.txt",
			map[string]string{"Content-Length": "5"}, []byte("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if request.Headers["content-length"] != "5" {
			t.Errorf("content-length = %q, want %q", request.Headers["content-length"], "5")
		}
	})

	t.Run("rejects a Content-Length that doesn't match the body", func(t *testing.T) {
		_, err := httpserver.NewRequest(httpserver.MethodPost, "/files/new.txt",
			map[string]string{"content-length": "999"}, []byte("hi"))
		if err == nil {
			t.Fatal("expected an error for a Content-Length that doesn't match the body")
		}
	})

	t.Run("computes Content-Length automatically when the body is non-empty and it's missing", func(t *testing.T) {
		request, err := httpserver.NewRequest(httpserver.MethodPost, "/files/new.txt", nil, []byte("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if request.Headers["content-length"] != "5" {
			t.Errorf("content-length = %q, want %q", request.Headers["content-length"], "5")
		}
	})

	t.Run("accepts an explicit Content-Length of 0 with an empty body", func(t *testing.T) {
		request, err := httpserver.NewRequest(httpserver.MethodPost, "/files/new.txt",
			map[string]string{"content-length": "0"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if request.Headers["content-length"] != "0" {
			t.Errorf("content-length = %q, want %q", request.Headers["content-length"], "0")
		}
	})

	t.Run("rejects methods other than GET and POST", func(t *testing.T) {
		methods := []httpserver.HTTPMethod{"PUT", "DELETE", "PATCH", "HEAD", "CONNECT", "OPTIONS", "TRACE", "get"}
		for _, method := range methods {
			t.Run(string(method), func(t *testing.T) {
				_, err := httpserver.NewRequest(method, "/files/abc.txt", nil, nil)
				if err == nil {
					t.Fatalf("expected an error for method %s", method)
				}
			})
		}
	})

	t.Run("rejects a path that isn't relative", func(t *testing.T) {
		paths := []string{
			"http://example.com/file.txt",
			"*",
			"file.txt",
			"",
		}
		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				_, err := httpserver.NewRequest(httpserver.MethodGet, path, nil, nil)
				if err == nil {
					t.Fatalf("expected an error for path %q", path)
				}
			})
		}
	})

	t.Run("rejects Transfer-Encoding: chunked", func(t *testing.T) {
		_, err := httpserver.NewRequest(httpserver.MethodPost, "/files/new.txt",
			map[string]string{"Transfer-Encoding": "chunked", "content-length": "5"}, []byte("hello"))
		if err == nil {
			t.Fatal("expected an error for chunked transfer encoding")
		}
	})
}

func TestParseRequest(t *testing.T) {
	t.Run("parses a GET request line", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET /file/abc.txt HTTP/1.1\r\n\r\n",
		))
		if err != nil {
			t.Fatal(err)
		}
		if request.Method != httpserver.MethodGet {
			t.Errorf("method = %q, want %q", request.Method, httpserver.MethodGet)
		}
		if request.Path != "/file/abc.txt" {
			t.Errorf("path = %q, want %q", request.Path, "/file/abc.txt")
		}
	})

	t.Run("parses a POST request line", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"POST /file/abc.txt HTTP/1.1\r\n\r\n",
		))
		if err != nil {
			t.Fatal(err)
		}
		if request.Method != httpserver.MethodPost {
			t.Errorf("method = %q, want %q", request.Method, httpserver.MethodPost)
		}
	})

	t.Run("rejects methods other than GET and POST", func(t *testing.T) {
		methods := []string{"PUT", "DELETE", "PATCH", "HEAD", "CONNECT", "OPTIONS", "TRACE", "BREW"}
		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				_, err := httpserver.ParseRequest(bytes.NewBufferString(
					fmt.Sprintf("%s /file/abc.txt HTTP/1.1\r\n\r\n", method),
				))
				if err == nil {
					t.Fatalf("expected an error for method %s", method)
				}
			})
		}
	})

	t.Run("rejects a lowercase method", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"get /file/abc.txt HTTP/1.1\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for a lowercase method")
		}
	})

	for _, protocol := range []string{"HTTP/1.0", "HTTP/2.0", "HTTP/1.1x"} {
		t.Run("rejects protocol "+protocol, func(t *testing.T) {
			_, err := httpserver.ParseRequest(bytes.NewBufferString(
				fmt.Sprintf("GET /file/abc.txt %s\r\n\r\n", protocol),
			))
			if err == nil {
				t.Fatalf("expected an error for protocol %s", protocol)
			}
		})
	}

	t.Run("accepts a relative path with a query string", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET /files?name=abc.txt HTTP/1.1\r\n\r\n",
		))
		if err != nil {
			t.Fatal(err)
		}
		if request.Path != "/files?name=abc.txt" {
			t.Errorf("path = %q, want %q", request.Path, "/files?name=abc.txt")
		}
	})

	t.Run("rejects an absolute-form target", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET http://example.com/file.txt HTTP/1.1\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for an absolute-form target")
		}
	})

	t.Run("rejects an asterisk-form target", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET * HTTP/1.1\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for an asterisk-form target")
		}
	})

	t.Run("rejects a path without a leading slash", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET file.txt HTTP/1.1\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for a path without a leading slash")
		}
	})

	t.Run("normalizes header names to lowercase", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET / HTTP/1.1\r\nHost: example.com\r\nX-Request-ID: abc123\r\n\r\n",
		))
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"host":         "example.com",
			"x-request-id": "abc123",
		}
		if len(request.Headers) != len(want) {
			t.Fatalf("headers = %#v, want %#v", request.Headers, want)
		}
		for name, value := range want {
			if request.Headers[name] != value {
				t.Errorf("header %q = %q, want %q", name, request.Headers[name], value)
			}
		}
	})

	// RFC 7230 3.2.4: hay que descartar el OWS alrededor del valor del header.
	t.Run("trims optional whitespace around header values", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET / HTTP/1.1\r\nHost:    example.com   \r\n\r\n",
		))
		if err != nil {
			t.Fatal(err)
		}
		if request.Headers["host"] != "example.com" {
			t.Errorf("host header = %q, want %q", request.Headers["host"], "example.com")
		}
	})

	t.Run("rejects a header line without a colon", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"GET / HTTP/1.1\r\nHost example.com\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for a malformed header line")
		}
	})

	t.Run("includes a body matching Content-Length", func(t *testing.T) {
		body := "hello"
		request, err := httpserver.ParseRequest(bytes.NewBufferString(fmt.Sprintf(
			"POST /files/new.txt HTTP/1.1\r\nContent-Length: %d\r\n\r\n%s",
			len(body), body,
		)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(request.Body, []byte(body)) {
			t.Errorf("body = %q, want %q", request.Body, body)
		}
	})

	t.Run("reads only Content-Length bytes when more data follows", func(t *testing.T) {
		request, err := httpserver.ParseRequest(bytes.NewBufferString(
			"POST /files/new.txt HTTP/1.1\r\nContent-Length: 5\r\n\r\nhelloEXTRA",
		))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(request.Body, []byte("hello")) {
			t.Errorf("body = %q, want %q", request.Body, "hello")
		}
	})

	// Solo cubre "el reader se quedó sin datos antes de completar Content-Length"
	// (io.ErrUnexpectedEOF). El caso "bloquea y tira error por deadline" no se
	// puede probar con un bytes.Buffer porque nunca bloquea — eso va en un test
	// de integración a nivel Serve/handleConn, con el deadline sobre el conn real.
	t.Run("errors when fewer bytes are available than Content-Length declares", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"POST /files/new.txt HTTP/1.1\r\nContent-Length: 20\r\n\r\nhello",
		))
		if err == nil {
			t.Fatal("expected an error when the body is shorter than Content-Length")
		}
	})

	t.Run("rejects chunked transfer encoding", func(t *testing.T) {
		_, err := httpserver.ParseRequest(bytes.NewBufferString(
			"POST /files/new.txt HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
		))
		if err == nil {
			t.Fatal("expected an error for chunked transfer encoding")
		}
	})
}
