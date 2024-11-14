package e5

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var once sync.Once
var e5RerankerClient *http.Client

func InitializeSingletonClient() (*http.Client, error) {
	var err error
	once.Do(func() {
		e5RerankerClient = &http.Client{
			Timeout: 10 * time.Second,
		}

		if err != nil {
			logrus.Errorf("error initializing e5 searcher client: %v", err)
		}

	})
	return e5RerankerClient, err
}

func GetE5RerankerClient() (*http.Client, error) {
	if e5RerankerClient == nil {
		return nil, errors.New("no singleton client")
	}
	return e5RerankerClient, nil
}
