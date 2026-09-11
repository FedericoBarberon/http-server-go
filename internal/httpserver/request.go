package httpserver

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Request struct {
	Method  HTTPMethod
	Path    string
	Headers map[string]string
	Body    []byte
}

func ParseRequest(r io.Reader) (Request, error) {
	var req Request
	buf := bufio.NewReader(r)

	method, path, err := parseReqLine(buf)
	if err != nil {
		return Request{}, err
	}

	req.Method = method
	req.Path = path

	req.Headers = make(map[string]string)
	for {
		key, val, done, err := parseHeader(buf)
		if err != nil {
			return Request{}, err
		}

		if done {
			break
		}

		req.Headers[key] = val
	}

	if val, ok := req.Headers["transfer-encoding"]; ok && strings.Contains(val, "chunked") {
		return Request{}, fmt.Errorf("header Transfer-Encoding: chunked not supported")
	}

	contentLengthStr, ok := req.Headers["content-length"]

	if !ok {
		return req, nil
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil || contentLength < 0 {
		return Request{}, fmt.Errorf("%s is not a valid Content-Length", contentLengthStr)
	}

	body, err := parseBody(buf, contentLength)
	if err != nil {
		return Request{}, err
	}

	req.Body = body
	return req, nil
}

func parseReqLine(r *bufio.Reader) (method HTTPMethod, path string, err error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read request")
	}
	line = strings.TrimSuffix(line, "\r\n")

	reqLine := strings.Split(line, " ")

	if len(reqLine) != 3 {
		return "", "", fmt.Errorf("%s is not a valid HTTP request line", reqLine)
	}

	method, err = HTTPMethodFromStr(reqLine[0])
	if err != nil {
		return "", "", fmt.Errorf("%s is not a valid HTTP method", reqLine[0])
	}

	path = reqLine[1]
	if !strings.HasPrefix(path, "/") {
		return "", "", fmt.Errorf("%s is not a supported path", path)
	}

	protocol := reqLine[2]
	if protocol != "HTTP/1.1" {
		return "", "", fmt.Errorf("%s is not a supported protocol", protocol)
	}

	return
}

func parseHeader(r *bufio.Reader) (key string, val string, done bool, err error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", "", false, fmt.Errorf("failed to read request")
	}

	if line == "\r\n" {
		return "", "", true, nil
	}

	header := strings.TrimSuffix(line, "\r\n")
	key, val, ok := strings.Cut(header, ":")
	if !ok {
		return "", "", false, fmt.Errorf("invalid header")
	}

	key = strings.ToLower(strings.TrimSpace(key))
	val = strings.TrimSpace(val)

	return
}

func parseBody(r *bufio.Reader, contentLength int) ([]byte, error) {
	bodyBuf := make([]byte, contentLength)
	n, err := io.ReadFull(r, bodyBuf)
	if err != nil {
		return nil, fmt.Errorf("expected %d bytes of body, got %d", contentLength, n)
	}

	return bodyBuf, nil
}
