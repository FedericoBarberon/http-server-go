package httpserver

import "fmt"

type HTTPMethod string

const (
	MethodGet  HTTPMethod = "GET"
	MethodPost HTTPMethod = "POST"
)

func HTTPMethodFromStr(s string) (HTTPMethod, error) {
	switch s {
	case "GET":
		return MethodGet, nil
	case "POST":
		return MethodPost, nil
	default:
		return "", fmt.Errorf("invalid HTTP method: %q", s)
	}
}
