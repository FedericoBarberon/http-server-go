package httpserver

import (
	"fmt"
	"net"
)

type Handle func(*Request) *Response

func Serve(l net.Listener, handle Handle) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("failed to accept connection")
			continue
		}

		go handleConn(conn, handle)
	}
}

func handleConn(conn net.Conn, handle Handle) {
	defer conn.Close()

	req, err := ParseRequest(conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	handle(&req)
}
