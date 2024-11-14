package response

import (
	"errors"
	"net/http"
)

func WriteStreamResponse(w http.ResponseWriter, b []byte) error {
	f, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}

	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
	_, _ = w.Write([]byte("data: " + string(b) + "\n\n"))
	f.Flush()

	return nil
}

func WriteStreamErrorResponse(w http.ResponseWriter, err error) error {
	f, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}

	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
	_, _ = w.Write([]byte("data: {\"error\": \"" + err.Error() + "\"}\n\n"))
	f.Flush()

	return nil
}
