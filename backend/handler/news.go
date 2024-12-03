package handler

import (
	"net/http"

	"bigkinds.or.kr/backend/service"
)

type NewsHandler struct {
	Service *service.NewsService
}

func (h *NewsHandler) GetNewsSummary(w http.ResponseWriter, r *http.Request) {

}
