package httpserver

var reasonPhrases = map[uint]string{
	200: "OK",
	204: "No Content",
	400: "Bad Request",
	404: "Not Found",
	405: "Method Not Allowed",
	500: "Internal Server Error",
	501: "Not Implemented",
}
