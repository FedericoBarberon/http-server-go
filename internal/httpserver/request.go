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

func NewRequest(method HTTPMethod, path string, headers map[string]string, body []byte) (Request, error) {
	if !method.IsValid() {
		return Request{}, fmt.Errorf("%s is not a supported HTTP method", method)
	}

	if !strings.HasPrefix(path, "/") {
		return Request{}, fmt.Errorf("%s is not a valid path", path)
	}

	normalizedHeaders := make(map[string]string)
	for k, v := range headers {
		normalizedKey := strings.TrimSpace(strings.ToLower(k))
		normalizedValue := strings.TrimSpace(v)

		if strings.ContainsAny(normalizedKey, ": \t\r\n") {
			return Request{}, fmt.Errorf("headers keys cannot contain spaces nor ':'")
		}

		if strings.ContainsAny(normalizedValue, "\n\r") {
			return Request{}, fmt.Errorf("header value cannot contain CR nor LF")
		}

		if _, ok := normalizedHeaders[normalizedKey]; ok {
			return Request{}, fmt.Errorf("found duplicated headers keys")
		}
		normalizedHeaders[normalizedKey] = strings.TrimSpace(v)
	}

	if val, ok := normalizedHeaders["transfer-encoding"]; ok && strings.Contains(val, "chunked") {
		return Request{}, fmt.Errorf("header Transfer-Encoding with chunked is not supported")
	}

	if contentLengthStr, ok := normalizedHeaders["content-length"]; ok {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return Request{}, fmt.Errorf("%s is not a valid Content-Length", contentLengthStr)
		}

		if contentLength != len(body) {
			return Request{}, fmt.Errorf("header Content-Length (%d) does not match body length (%d)", contentLength, len(body))
		}
	} else if len(body) > 0 {
		normalizedHeaders["content-length"] = strconv.Itoa(len(body))
	}

	return Request{Method: method, Path: path, Headers: normalizedHeaders, Body: body}, nil
}

func ParseRequest(r io.Reader) (Request, error) {
	buf := bufio.NewReader(r)

	method, path, err := parseReqLine(buf)
	if err != nil {
		return Request{}, err
	}

	headers := make(map[string]string)
	for {
		key, val, done, err := parseHeader(buf)
		if err != nil {
			return Request{}, err
		}

		if done {
			break
		}

		normalizedKey := strings.TrimSpace(strings.ToLower(key))

		if _, ok := headers[normalizedKey]; ok {
			return Request{}, fmt.Errorf("found duplicated headers keys")
		}
		headers[normalizedKey] = strings.TrimSpace(val)
	}

	var body []byte

	contentLengthStr, ok := headers["content-length"]

	if !ok {
		req, err := NewRequest(method, path, headers, body)
		if err != nil {
			return Request{}, err
		}

		return req, nil
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil || contentLength < 0 {
		return Request{}, fmt.Errorf("%s is not a valid Content-Length", contentLengthStr)
	}

	body, err = parseBody(buf, contentLength)
	if err != nil {
		return Request{}, err
	}

	req, err := NewRequest(method, path, headers, body)
	if err != nil {
		return Request{}, err
	}

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

	method = HTTPMethod(reqLine[0])
	path = reqLine[1]

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
