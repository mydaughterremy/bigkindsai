package opensearch

import (
	"sync"

	"bigkinds.or.kr/pkg/encoder"
	"bigkinds.or.kr/search/service/searcher/opensearch/query_strategy"
	"github.com/sirupsen/logrus"
)

var (
	queryStrategyProviderOnce sync.Once
	queryStrategyProvider     Provider[query_strategy.QueryStrategy]
	err                       error
)

type Provider[T any] interface {
	Add(name string, queryStrategy T)
	Get(name string) T
}

type QueryStrategyProvider[T any] struct {
	quertStrategies map[string]T
	mux             *sync.Mutex
}

func (p *QueryStrategyProvider[T]) Add(queryStrategyName string, queryStrategy T) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.quertStrategies[queryStrategyName] = queryStrategy
}

func (p *QueryStrategyProvider[T]) Get(queryStrategyName string) T {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.quertStrategies[queryStrategyName]
}

func NewQueryStrategyProvider() (Provider[query_strategy.QueryStrategy], error) {
	queryStrategyProviderOnce.Do(func() {
		queryStrategyProvider = &QueryStrategyProvider[query_strategy.QueryStrategy]{
			quertStrategies: make(map[string]query_strategy.QueryStrategy),
			mux:             &sync.Mutex{},
		}
		// add text query strategy
		textQueryStrategy := query_strategy.NewTextQueryStrategy()
		queryStrategyProvider.Add("text", textQueryStrategy)

		// add vector query strategy
		sentenceTransformers := encoder.NewSentenceTransformers() //TODO: replace this with encoder provider
		queryStrategyProvider.Add("vector", query_strategy.NewVectorQueryStrategy(sentenceTransformers))

		logrus.Debugf("query strategy provider is created. %v", queryStrategyProvider)
	})

	if err != nil {
		return nil, err
	}
	return queryStrategyProvider, nil
}
