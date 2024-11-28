package search_engine

import (
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/sirupsen/logrus"
)

var once sync.Once
var openSearchClient *opensearch.Client

// opensearch 수정
func InitializeSingletonOpenSearchClient(addressString, username, password string) (*opensearch.Client, error) {
	var err error
	once.Do(func() {
		addresses := strings.Split(addressString, ",")

		openSearchClient, err = opensearch.NewClient(opensearch.Config{
			Addresses: addresses,
			Username:  username,
			Password:  password,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 60 * time.Second,
				TLSHandshakeTimeout:   1 * time.Second,
				MaxIdleConnsPerHost:   128,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		})

		if err != nil {
			logrus.Errorf("error initializing opensearch client: %v", err)
		}

	})
	return openSearchClient, err
}

func GetOpenSearchClient() (*opensearch.Client, error) {
	if openSearchClient == nil {
		return nil, errors.New("no singleton client")
	}
	return openSearchClient, nil
}
