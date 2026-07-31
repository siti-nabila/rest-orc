package fixtures

import "net/http"

func Response(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}
}
