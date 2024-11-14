package main

import (
	"time"

	"bigkinds.or.kr/backend/service"
)

func NewQAService(timezone *time.Location) (*service.QAService, error) {
	db, err := GetGORMDB()
	if err != nil {
		return nil, err
	}

	return service.NewQAService(db, timezone), nil
}
