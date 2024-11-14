package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/service"
)

type StatisticsHandler struct {
	StatisticsService *service.StatisticsService
}

func (h StatisticsHandler) prepareStatisticsRequestParameter(r *http.Request) (*time.Time, *time.Time, error) {
	fromParam := r.URL.Query().Get("from")
	if fromParam == "" {
		return nil, nil, errors.New("`from` is required")
	}
	from, err := time.Parse(time.DateOnly, fromParam)
	if err != nil {
		return nil, nil, errors.New("`from` is not a expected datetime format ('yyyy-MM-dd')")
	}

	toParam := r.URL.Query().Get("to")
	if toParam == "" {
		return nil, nil, errors.New("`to` is required")
	}
	to, err := time.Parse(time.DateOnly, toParam)
	if err != nil {
		return nil, nil, errors.New("`to` is not a expected datetime format ('yyyy-MM-dd')")
	}

	if from.After(to) {
		return nil, nil, errors.New("`from` should be before `to`")
	}

	to = to.Add(24 * time.Hour).Add(-1 * time.Nanosecond)

	return &from, &to, nil
}

func (h StatisticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	from, to, err := h.prepareStatisticsRequestParameter(r)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = service.PeriodUnitDay
	}
	if unit != service.PeriodUnitMonth && unit != service.PeriodUnitDay {
		w.WriteHeader(http.StatusBadRequest)
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, fmt.Errorf("`unit` should be one of ['MONTH', 'DAY']"))
		return
	}

	stats, err := h.StatisticsService.GetStatistics(ctx, from, to, unit)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, stats)
}
