package httpserver

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

type Response struct {
	Status  uint
	Headers map[string]string
	Body    []byte
}

func NewResponse(status uint, headers map[string]string, body []byte) (Response, error) {
	if status < 100 || status > 599 {
		return Response{}, fmt.Errorf("%d is not a valid HTTP status code", status)
	}

	normalizedHeaders := make(map[string]string)
	for k, v := range headers {
		normalizedKey := normalizeHeaderKey(k)
		normalizedValue := strings.TrimSpace(v)

		if strings.ContainsAny(normalizedKey, ": \t\r\n") {
			return Response{}, fmt.Errorf("headers keys cannot contain spaces nor ':'")
		}

		if strings.ContainsAny(normalizedValue, "\r\n") {
			return Response{}, fmt.Errorf("headers value cannot contain CR nor LF")
		}

		if _, ok := normalizedHeaders[normalizedKey]; ok {
			return Response{}, fmt.Errorf("found duplicated headers keys")
		}
		normalizedHeaders[normalizedKey] = normalizedValue
	}

	if val, ok := normalizedHeaders["Transfer-Encoding"]; ok && strings.Contains(val, "chunked") {
		return Response{}, fmt.Errorf("header Transfer-Encoding with chunked is not supported")
	}

	if contentLengthStr, ok := normalizedHeaders["Content-Length"]; ok {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil || contentLength < 0 {
			return Response{}, fmt.Errorf("%s is not a valid Content-Length", contentLengthStr)
		}

		if contentLength != len(body) {
			return Response{}, fmt.Errorf("header Content-Length (%d) does not match body length (%d)", contentLength, len(body))
		}
	} else {
		normalizedHeaders["Content-Length"] = strconv.Itoa(len(body))
	}

	return Response{Status: status, Headers: normalizedHeaders, Body: body}, nil
}

func (r *Response) Serialize() []byte {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.Status, r.ReasonPhrase())

	sortedKeys := slices.Sorted(maps.Keys(r.Headers))

	var buf bytes.Buffer
	buf.Grow(len(statusLine) + (len(sortedKeys) * 30) + len(r.Body))

	buf.WriteString(statusLine)

	for _, key := range sortedKeys {
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(r.Headers[key])
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")
	buf.Write(r.Body)

	return buf.Bytes()
}

func (r *Response) ReasonPhrase() string {
	return reasonPhrases[r.Status]
}

func normalizeHeaderKey(s string) string {
	s = strings.TrimSpace(s)
	normalizedKey := make([]byte, len(s))

	for i := range s {
		if i == 0 || s[i-1] == '-' {
			normalizedKey[i] = toUpperASCII(s[i])
		} else {
			normalizedKey[i] = toLowerASCII(s[i])
		}
	}

	return string(normalizedKey)
}

func toUpperASCII(c byte) byte {
	if 'a' <= c && c <= 'z' {
		return c - ('a' - 'A')
	}

	return c
}

func toLowerASCII(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + ('a' - 'A')
	}

	return c
}
