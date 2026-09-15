package httpserver

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type Handle func(Request) Response

func Serve(l net.Listener, handle Handle) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			fmt.Println("failed to accept connection")
			continue
		}

		go handleConn(conn, handle)
	}
}

func handleConn(conn net.Conn, handle Handle) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	req, err := ParseRequest(conn)
	if err != nil {
		fmt.Println(err)
		res, err := NewResponse(400, map[string]string{}, nil)
		if err != nil {
			fmt.Println("failed to build 400 response: ", err)
			return
		}

		conn.Write(res.Serialize())
		return
	}

	res := handle(req)
	conn.Write(res.Serialize())
}
