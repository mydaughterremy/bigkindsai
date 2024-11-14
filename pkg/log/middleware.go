package log

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ContextLoggerMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = WithLogger(ctx, logger)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func RequestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger, err := GetLogger(ctx)
		if err != nil {
			logger, _ = zap.NewProduction()
			ctx = WithLogger(ctx, logger)
			r = r.WithContext(ctx)
		}

		fields, err := NewRequestLogFieldFromHTTPRequest(r)
		if err != nil {
			logger.Error("Error while parsing request log field", zap.Any("error", err))
		}
		fields.RequestID = GetRequestIDFromContext(ctx)
		if fields.RequestID == "" {
			fields.RequestID = "bigkindsai-" + uuid.New().String()
			r.Header.Set("X-BigKindsAI-Request-Id", fields.RequestID) // if request id is not set, set it to request header
		}
		if fields.Path != "/healthz" {
			logger.Info("http request", fields.ZapField())
		}
		next.ServeHTTP(w, r)
	})
}

func calculateMS(ns int64) float64 {
	return float64(ns) / 1000000
}

func ResponseLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeHeader := true
		t := time.Now()
		requestId := GetRequestIDFromHTTPRequest(r)
		ctx := WithRequestID(r.Context(), requestId)
		r = r.WithContext(ctx)

		ctx = r.Context()
		logger, err := GetLogger(ctx)
		if err != nil {
			logger, _ = zap.NewProduction()
			logger.Warn("logger has been initialized while logging response")
		}

		interceptor := &FlushInterceptor{
			rw:       w,
			Recorder: httptest.NewRecorder(),
			flushFunc: func(w http.ResponseWriter, recorder *httptest.ResponseRecorder) {
				f, ok := w.(http.Flusher)
				if !ok {
					return
				}

				ms := calculateMS(time.Since(t).Nanoseconds())
				response := recorder.Result()
				writeResponseToResponseWriter(response, w, &writeHeader)
				f.Flush()
				printResponseLog(r, logger, requestId, response, ms)
			},
		}
		next.ServeHTTP(interceptor, r)
		ms := calculateMS(time.Since(t).Nanoseconds())
		writeResponseToResponseWriter(interceptor.Recorder.Result(), w, &writeHeader)
		if r.URL.Path != "/healthz" {
			printResponseLog(r, logger, requestId, interceptor.Recorder.Result(), ms)
		}
	})
}

func writeResponseToResponseWriter(response *http.Response, w http.ResponseWriter, writeHeader *bool) {
	if *writeHeader {
		for key := range response.Header {
			w.Header().Set(key, response.Header.Get(key))
		}
		w.WriteHeader(response.StatusCode)

		*writeHeader = false
	}

	buf, _ := io.ReadAll(response.Body)
	response.Body = io.NopCloser(bytes.NewReader(buf))
	_, _ = w.Write(buf)
}

func printResponseLog(r *http.Request, logger *zap.Logger, requestId string, response *http.Response, t float64) {
	fields, err := NewResponseLogFieldFromHTTPResponse(response, t)
	if err != nil {
		logger.Error("Error while parsing response log field", zap.Any("error", err))
	}

	fields.RequestID = requestId

	//logger.Info("http response", fields.ZapField())
}
