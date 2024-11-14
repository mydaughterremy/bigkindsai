package response

import (
	"encoding/json"
	"net/http"
)

func WriteJsonResponse(w http.ResponseWriter, r *http.Request, code int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(data)
}

func WriteJsonErrorResponse(w http.ResponseWriter, r *http.Request, code int, err error) error {
	return WriteJsonResponse(w, r, code, map[string]string{
		"error": err.Error(),
	})
}
