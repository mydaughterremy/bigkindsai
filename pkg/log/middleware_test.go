package log

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type mockHTTPHandler struct {
	Func func(w http.ResponseWriter, r *http.Request)
}

func (h mockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Func(w, r)
}

func TestContextLoggerMiddleware(t *testing.T) {
	logger, _ := zap.NewProduction()

	middleware := ContextLoggerMiddleware(logger, mockHTTPHandler{
		Func: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxLogger, err := GetLogger(ctx)
			assert.Nil(t, err)
			assert.Equal(t, logger, ctxLogger)
		},
	})

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", bytes.NewReader([]byte("")))
	middleware.ServeHTTP(resp, req)
}

func TestRequestLogMiddleware(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", bytes.NewReader([]byte("{\"test\":\"test\"}")))
	req = req.WithContext(WithRequestID(req.Context(), "test-request-id"))
	reqField, _ := NewRequestLogFieldFromHTTPRequest(req)
	reqField.RequestID = "test-request-id"
	expectedZapField := reqField.ZapField()
	recorder := httptest.NewRecorder()

	observer, logs := observer.New(zap.InfoLevel)
	logger := zap.New(observer)

	middleware := ContextLoggerMiddleware(logger, RequestLogMiddleware(mockHTTPHandler{
		Func: func(w http.ResponseWriter, r *http.Request) {
			entries := logs.All()
			assert.Len(t, entries, 1)

			ctxFields := entries[0].Context
			assert.Len(t, ctxFields, 1)
			assert.Equal(t, expectedZapField, ctxFields[0])
		},
	}))

	middleware.ServeHTTP(recorder, req)
}

func TestResponseLogMiddleware(t *testing.T) {
	expectedBody := []byte("{\"test\":\"test\"}")
	expectedStatusCode := 999
	headerKey := "Upstage-Test"
	expectedHeaderValue := "Test"

	req, _ := http.NewRequest("GET", "/", bytes.NewReader([]byte("")))
	header := req.Header
	header.Set("X-Upstage-Request-Id", "test-request-id")

	observer, logs := observer.New(zap.InfoLevel)
	logger := zap.New(observer)
	middleware := ContextLoggerMiddleware(logger, ResponseLogMiddleware(mockHTTPHandler{
		Func: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(headerKey, expectedHeaderValue)
			w.WriteHeader(expectedStatusCode)
			_, _ = w.Write(expectedBody)
		},
	}))

	recorder := httptest.NewRecorder()
	middleware.ServeHTTP(recorder, req)
	assert.Equal(t, bytes.NewBuffer(expectedBody), recorder.Body)                // test if response body is not modified
	assert.Equal(t, expectedStatusCode, recorder.Code)                           // test if response code is not modified
	assert.EqualValues(t, expectedHeaderValue, recorder.Header().Get(headerKey)) // test if response header is not modified

	entries := logs.All()
	assert.Len(t, entries, 1)

	ctxFields := entries[0].Context
	assert.Len(t, ctxFields, 1)

	ms := 1.0
	respField, _ := NewResponseLogFieldFromHTTPResponse(&http.Response{
		Body:       io.NopCloser(bytes.NewReader(expectedBody)),
		StatusCode: expectedStatusCode,
	}, ms)
	respField.RequestID = "test-request-id"

	// hack to ignore response time checking
	_interface, _ := ctxFields[0].Interface.(*responseLogField)
	_interface.TimeMS = ms
	ctxFields[0].Interface = _interface

	expectedZapField := respField.ZapField()
	assert.Equal(t, expectedZapField, ctxFields[0])
}

func TestCalculateMS(t *testing.T) {
	assert.Equal(t, float64(1000), calculateMS(1000000000))
}
