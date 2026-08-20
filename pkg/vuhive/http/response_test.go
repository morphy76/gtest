package http_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vuhivehttp "github.com/morphy76/vuhive/pkg/vuhive/http"
)

func TestResponse_Text_ReturnsBodyAsString(t *testing.T) {
	resp := vuhivehttp.Response{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("hello world"),
	}

	assert.Equal(t, "hello world", resp.Text())
}

func TestResponse_Text_EmptyBody(t *testing.T) {
	resp := vuhivehttp.Response{
		StatusCode: http.StatusNoContent,
		Body:       nil,
	}

	assert.Equal(t, "", resp.Text())
}

func TestResponse_JSON_UnmarshalSuccess(t *testing.T) {
	type payload struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
	}

	resp := vuhivehttp.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"status":"ok","code":200}`),
	}

	var p payload
	err := resp.JSON(&p)

	require.NoError(t, err)
	assert.Equal(t, "ok", p.Status)
	assert.Equal(t, 200, p.Code)
}

func TestResponse_JSON_UnmarshalError(t *testing.T) {
	resp := vuhivehttp.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{invalid json}`),
	}

	var target map[string]any
	err := resp.JSON(&target)

	assert.Error(t, err, "invalid JSON should return an error")
}

func TestResponse_JSON_NilBody(t *testing.T) {
	resp := vuhivehttp.Response{
		StatusCode: http.StatusOK,
		Body:       nil,
	}

	var target map[string]any
	err := resp.JSON(&target)

	assert.Error(t, err, "nil body should return an error")
}
