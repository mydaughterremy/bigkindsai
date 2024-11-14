package utils

import "time"

type CurrentTime struct {
	Location *time.Location
	Time     time.Time
}

func GetCurrentKSTTime() (CurrentTime, error) {
	// get current time
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return CurrentTime{}, err
	} else {
		return CurrentTime{
			Location: location,
			Time:     time.Now().In(location),
		}, nil
	}
}

func GetMockCurrentKSTTime() (CurrentTime, error) {
	// get current time
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return CurrentTime{}, err
	} else {
		return CurrentTime{
			Location: location,
			Time:     time.Date(time.Now().Year(), time.July, 31, 0, 0, 0, 0, location).In(location),
		}, nil
	}
}
