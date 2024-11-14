package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type qaHandler struct {
	service *service.QAService
}

type ListQAsWithPaginationParam struct {
	From        time.Time
	To          time.Time
	Offset      int
	Limit       int
	SearchQuery string
}

func (h *qaHandler) parseListQAswithPaginationRequestParam(r *http.Request) (*ListQAsWithPaginationParam, error) {
	var from, to time.Time
	var err error
	fromParam := r.URL.Query().Get("from")
	if fromParam != "" {
		from, err = time.Parse(time.DateOnly, fromParam)
		if err != nil {
			return nil, errors.New("`from` is not a expected datetime format ('yyyy-MM-dd')")
		}
	}
	toParam := r.URL.Query().Get("to")
	if toParam != "" {
		to, err = time.Parse(time.DateOnly, toParam)
		if err != nil {
			return nil, errors.New("`to` is not a expected datetime format ('yyyy-MM-dd')")
		}
	}

	if !from.IsZero() && !to.IsZero() {
		if from.After(to) {
			return nil, errors.New("`from` should be before `to`")
		}
	}

	if !to.IsZero() {
		to = to.Add(24 * time.Hour).Add(-1 * time.Nanosecond)
	}

	offsetParam := r.URL.Query().Get("offset")
	if offsetParam == "" {
		offsetParam = "0"
	}
	offset, err := strconv.Atoi(offsetParam)
	if err != nil {
		return nil, errors.New("`offset` is not a number")
	}
	if offset < 0 {
		return nil, errors.New("`offset` should be positive")
	}

	limitParam := r.URL.Query().Get("limit")
	if limitParam == "" {
		limitParam = "10"
	}
	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		return nil, errors.New("`limit` is not a number")
	}
	if limit < 0 {
		return nil, errors.New("`limit` should be positive")
	}

	listQAParam := &ListQAsWithPaginationParam{
		From:   from,
		To:     to,
		Offset: offset,
		Limit:  limit,
	}

	searchQueryParam := r.URL.Query().Get("search_query")
	if searchQueryParam != "" {
		listQAParam.SearchQuery = searchQueryParam
	}
	return listQAParam, nil
}

func (h *qaHandler) GetQA(w http.ResponseWriter, r *http.Request) {
	type GetQAResponse struct {
		QA *model.QA `json:"qa"`
	}
	ctx := r.Context()
	qaID := chi.URLParam(r, "qa_id")
	qa, err := h.service.GetQA(ctx, qaID)
	getQAResponse := GetQAResponse{
		QA: qa,
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusOK, getQAResponse)
}

func (h *qaHandler) DeleteQA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qaId := chi.URLParam(r, "qa_id")
	err := h.service.DeleteQA(ctx, qaId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}

type DeleteQAsRequest struct {
	Ids []string `json:"ids"`
}

func (h *qaHandler) DeleteQAs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req DeleteQAsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	err = h.service.DeleteQAs(ctx, req.Ids)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}

	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}

func (h *qaHandler) ListQAsWithPagination(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params, err := h.parseListQAswithPaginationRequestParam(r)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}
	qa, err := h.service.ListQAsWithPagination(ctx, params.From, params.To, params.SearchQuery, params.Offset, params.Limit)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	_ = response.WriteJsonResponse(w, r, http.StatusOK, qa)
}

func (h *qaHandler) GetVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qaID := chi.URLParam(r, "qa_id")
	vote, err := h.service.GetVote(ctx, qaID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusOK, map[string]string{
		"vote": vote,
	})
}

func (h *qaHandler) UpvoteQA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qaID := chi.URLParam(r, "qa_id")
	err := h.service.UpvoteQA(ctx, qaID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}

func (h *qaHandler) DownvoteQA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qaID := chi.URLParam(r, "qa_id")
	err := h.service.DownvoteQA(ctx, qaID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}

func (h *qaHandler) DeleteVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qaID := chi.URLParam(r, "qa_id")
	err := h.service.DeleteVote(ctx, qaID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}
