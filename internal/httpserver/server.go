package httpserver

import (
	"bufio"
	"fmt"
	"net"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection")
			continue
		}

		handle(conn)
		conn.Close()
	}
}

func handle(conn net.Conn) {
	buf := bufio.NewReader(conn)

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			fmt.Println("Failed to read request: ", err)
			return
		}

		if line == "\r\n" {
			break
		}
	}

	resp := []byte("HTTP/1.1 200 OK")
	if _, err := conn.Write(resp); err != nil {
		fmt.Println("Failed to send response: ", err)
	}
}
