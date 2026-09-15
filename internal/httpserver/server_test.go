package httpserver_test

import (
	"bytes"
	"http-server/internal/httpserver"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	t.Run("returns nil when the listener is closed", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan error, 1)
		go func() {
			done <- httpserver.Serve(l, func(httpserver.Request) httpserver.Response {
				return httpserver.Response{}
			})
		}()

		if err := l.Close(); err != nil {
			t.Fatal(err)
		}

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve() = %v, want nil after the listener is closed", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Serve did not return after the listener was closed")
		}
	})

	t.Run("calls the handler and writes its response for a valid request", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		// Un channel en vez de una variable compartida: es la forma correcta
		// de sincronizar un valor que se escribe en la goroutine de Serve y
		// se lee en la del test — una variable simple no te garantiza el
		// happens-before necesario según el memory model de Go.
		requests := make(chan httpserver.Request, 1)
		handle := func(req httpserver.Request) httpserver.Response {
			requests <- req
			response, _ := httpserver.NewResponse(200, nil, []byte("hello"))
			return response
		}
		go httpserver.Serve(l, handle)

		conn, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		if _, err := conn.Write([]byte("GET /files/abc.txt HTTP/1.1\r\n\r\n")); err != nil {
			t.Fatal(err)
		}

		got, err := io.ReadAll(conn)
		if err != nil {
			t.Fatal(err)
		}

		want := "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"
		if string(got) != want {
			t.Errorf("response = %q, want %q", got, want)
		}

		select {
		case req := <-requests:
			if req.Path != "/files/abc.txt" {
				t.Errorf("handler received path %q, want %q", req.Path, "/files/abc.txt")
			}
		default:
			t.Error("handle was not called")
		}
	})

	t.Run("returns a 400 response when the request fails to parse", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		var handleCalled atomic.Bool
		go httpserver.Serve(l, func(httpserver.Request) httpserver.Response {
			handleCalled.Store(true)
			return httpserver.Response{}
		})

		conn, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		if _, err := conn.Write([]byte("GET / HTTP/2.0\r\n\r\n")); err != nil {
			t.Fatal(err)
		}

		got, err := io.ReadAll(conn)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.HasPrefix(got, []byte("HTTP/1.1 400")) {
			t.Errorf("response = %q, want it to start with %q", got, "HTTP/1.1 400")
		}
		if handleCalled.Load() {
			t.Error("handle should not be called for a request that fails to parse")
		}
	})

	// Asunción: una conexión = un solo request por ahora. Si más adelante
	// sumás conexiones persistentes, este test deja de tener sentido tal
	// cual está — pasaría a esperar un segundo request en vez de un EOF.
	t.Run("closes the connection after writing the response", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		handle := func(httpserver.Request) httpserver.Response {
			response, _ := httpserver.NewResponse(204, nil, nil)
			return response
		}
		go httpserver.Serve(l, handle)

		conn, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		io.ReadAll(conn) // agota la response

		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, err = conn.Read(make([]byte, 1))
		if err != io.EOF {
			t.Errorf("Read() error = %v, want io.EOF (se esperaba que el servidor cerrara la conexión)", err)
		}
	})
}
