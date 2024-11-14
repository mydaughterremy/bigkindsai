package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheckMiddleware(t *testing.T) {
	httpHandler := http.NewServeMux()
	httpHandler.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpHandlerWithMiddleware := HttpHealthcheckMiddleware(httpHandler, "/healthcheck")

	req := http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/healthcheck"},
	}
	recorder := httptest.NewRecorder()

	httpHandlerWithMiddleware.ServeHTTP(recorder, &req)

	assert.Equal(t, recorder.Code, http.StatusOK)
}
