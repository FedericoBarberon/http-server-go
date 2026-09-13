package httpserver_test

import (
	"fmt"
	"http-server/internal/httpserver"
	"testing"
)

func TestNewResponse(t *testing.T) {
	t.Run("creates a valid response with no body", func(t *testing.T) {
		response, err := httpserver.NewResponse(204, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if response.Status != 204 {
			t.Errorf("status = %d, want %d", response.Status, 204)
		}
		if response.Headers["Content-Length"] != "0" {
			t.Errorf("content-length = %q, want %q", response.Headers["Content-Length"], "0")
		}
	})

	t.Run("accepts boundary status codes", func(t *testing.T) {
		for _, status := range []int{100, 200, 404, 500, 599} {
			t.Run(fmt.Sprintf("%d", status), func(t *testing.T) {
				_, err := httpserver.NewResponse(status, nil, nil)
				if err != nil {
					t.Errorf("unexpected error for status %d: %v", status, err)
				}
			})
		}
	})

	t.Run("rejects status codes outside 100-599", func(t *testing.T) {
		for _, status := range []int{0, 99, 600, 999} {
			t.Run(fmt.Sprintf("%d", status), func(t *testing.T) {
				_, err := httpserver.NewResponse(status, nil, nil)
				if err == nil {
					t.Errorf("expected an error for status %d", status)
				}
			})
		}
	})

	t.Run("normalizes header keys to Title-Case", func(t *testing.T) {
		response, err := httpserver.NewResponse(200,
			map[string]string{"content-type": "text/plain", "x-request-id": "abc123"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"Content-Type":   "text/plain",
			"X-Request-Id":   "abc123",
			"Content-Length": "0",
		}
		if len(response.Headers) != len(want) {
			t.Fatalf("headers = %#v, want %#v", response.Headers, want)
		}
		for key, value := range want {
			if response.Headers[key] != value {
				t.Errorf("header %q = %q, want %q", key, response.Headers[key], value)
			}
		}
	})

	t.Run("rejects a header key containing a colon", func(t *testing.T) {
		_, err := httpserver.NewResponse(200, map[string]string{"x-request:id": "abc"}, nil)
		if err == nil {
			t.Fatal("expected an error for a header key containing a colon")
		}
	})

	t.Run("rejects a header key containing a space", func(t *testing.T) {
		_, err := httpserver.NewResponse(200, map[string]string{"x request id": "abc"}, nil)
		if err == nil {
			t.Fatal("expected an error for a header key containing a space")
		}
	})

	// Mismo problema que encontramos en NewRequest: dos keys que colisionan
	// después de normalizar a Title-Case.
	t.Run("rejects headers that collide after normalization", func(t *testing.T) {
		_, err := httpserver.NewResponse(200,
			map[string]string{"Content-Type": "text/plain", "content-type": "text/html"}, nil)
		if err == nil {
			t.Fatal("expected an error for headers colliding after normalization")
		}
	})

	t.Run("accepts a Content-Length that matches the body", func(t *testing.T) {
		response, err := httpserver.NewResponse(200,
			map[string]string{"content-length": "5"}, []byte("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if response.Headers["Content-Length"] != "5" {
			t.Errorf("content-length = %q, want %q", response.Headers["Content-Length"], "5")
		}
	})

	t.Run("rejects a Content-Length that doesn't match the body", func(t *testing.T) {
		_, err := httpserver.NewResponse(200,
			map[string]string{"content-length": "999"}, []byte("hi"))
		if err == nil {
			t.Fatal("expected an error for a Content-Length that doesn't match the body")
		}
	})

	t.Run("computes Content-Length automatically when the body is non-empty and it's missing", func(t *testing.T) {
		response, err := httpserver.NewResponse(200, nil, []byte("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if response.Headers["Content-Length"] != "5" {
			t.Errorf("content-length = %q, want %q", response.Headers["Content-Length"], "5")
		}
	})

	t.Run("rejects Transfer-Encoding: chunked", func(t *testing.T) {
		_, err := httpserver.NewResponse(200,
			map[string]string{"transfer-encoding": "chunked", "content-length": "5"}, []byte("hello"))
		if err == nil {
			t.Fatal("expected an error for chunked transfer encoding")
		}
	})

	t.Run("rejects a header key containing a bare CR, LF, or tab", func(t *testing.T) {
		keys := []string{"x-foo\rbar", "x-foo\nbar", "x-foo\tbar"}
		for _, key := range keys {
			t.Run(key, func(t *testing.T) {
				_, err := httpserver.NewResponse(200, map[string]string{key: "value"}, nil)
				if err == nil {
					t.Fatalf("expected an error for header key %q", key)
				}
			})
		}
	})

	t.Run("rejects a header value containing CR or LF", func(t *testing.T) {
		values := []string{
			"text/plain\rSet-Cookie: evil=true",
			"text/plain\nSet-Cookie: evil=true",
			"text/plain\r\nSet-Cookie: evil=true",
		}
		for _, value := range values {
			t.Run(value, func(t *testing.T) {
				_, err := httpserver.NewResponse(200,
					map[string]string{"content-type": value}, nil)
				if err == nil {
					t.Fatalf("expected an error for header value %q", value)
				}
			})
		}
	})

	t.Run("normalizes a fully uppercase header key to Title-Case", func(t *testing.T) {
		response, err := httpserver.NewResponse(200, map[string]string{"HOST": "example.com"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := response.Headers["Host"]; !ok {
			t.Errorf("headers = %#v, expected a canonical \"Host\" key", response.Headers)
		}
	})
}
