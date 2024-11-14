package log

import (
	"net/http"
	"net/http/httptest"
)

type FlushInterceptor struct {
	rw       http.ResponseWriter
	Recorder *httptest.ResponseRecorder

	flushFunc func(w http.ResponseWriter, recorder *httptest.ResponseRecorder)
}

func (r *FlushInterceptor) Header() http.Header {
	return r.Recorder.Header()
}

func (r *FlushInterceptor) Write(buf []byte) (int, error) {
	return r.Recorder.Write(buf)
}

func (r *FlushInterceptor) WriteHeader(code int) {
	r.Recorder.WriteHeader(code)
}

func (r *FlushInterceptor) Flush() {
	r.flushFunc(r.rw, r.Recorder)
	r.Recorder = httptest.NewRecorder()
}
