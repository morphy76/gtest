package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Response wraps the result of an HTTP request with convenience accessors.
// The response body is eagerly read and the underlying http.Response.Body is closed
// before this struct is returned, so callers do not need to manage body lifecycle.
type Response struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Headers contains the response headers.
	Headers http.Header

	// Body contains the raw response body bytes, eagerly read and closed.
	Body []byte
}

// Text returns the response body as a string.
func (r *Response) Text() string {
	return string(r.Body)
}

// JSON unmarshals the response body into the provided target value.
// Returns an error if the body is nil or cannot be parsed as JSON.
func (r *Response) JSON(v any) error {
	if r.Body == nil {
		return errors.New("vuhive/http: response body is nil")
	}
	return json.Unmarshal(r.Body, v)
}
