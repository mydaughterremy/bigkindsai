package provider

import (
	"sync"

	pb "bigkinds.or.kr/proto/searcher"
	"bigkinds.or.kr/search/service/searcher"
	"bigkinds.or.kr/search/service/searcher/opensearch"
	"bigkinds.or.kr/search/service/searcher/opensearch/query_strategy"
	"github.com/sirupsen/logrus"
)

var (
	searcherProviderOnce sync.Once
	searcherProvider     Provider[searcher.Searcher]
	err                  error
)

type Provider[T any] interface {
	Add(name string, searcher T)
	Get(name string) T
}

type SearcherProvider[T any] struct {
	searchers map[string]T
	mux       *sync.Mutex
}

func (p *SearcherProvider[T]) Add(searcherName string, searcher T) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.searchers[searcherName] = searcher
}

func (p *SearcherProvider[T]) Get(searcherName string) T {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.searchers[searcherName]
}

func NewSearcherProvider(provider opensearch.Provider[query_strategy.QueryStrategy], config *pb.SearcherConfig) (Provider[searcher.Searcher], error) {
	searcherProviderOnce.Do(func() {
		searcherProvider = &SearcherProvider[searcher.Searcher]{
			searchers: make(map[string]searcher.Searcher),
			mux:       &sync.Mutex{},
		}
		opensearch, err := searcher.NewOpenSearchSearcher(
			provider,
			config,
		)
		if err != nil {
			logrus.Errorf("failed to create opensearch client. %v", err)
		}

		searcherProvider.Add("opensearch", opensearch)

		logrus.Debugf("searcher provider is created. %v", searcherProvider)
	})

	if err != nil {
		return nil, err
	}
	return searcherProvider, nil
}
