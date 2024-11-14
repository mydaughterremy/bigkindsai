package provider

import (
	"sync"

	"bigkinds.or.kr/search/service/reranker"
	"github.com/sirupsen/logrus"
)

var (
	rerankerProviderOnce sync.Once
	rerankerProvider     Provider[reranker.Reranker]
)

type RerankerProvider[T any] struct {
	rerankers map[string]T
	mux       *sync.Mutex
}

func (p *RerankerProvider[T]) Add(rerankerName string, reranker T) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.rerankers[rerankerName] = reranker
}

func (p *RerankerProvider[T]) Get(rerankerName string) T {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.rerankers[rerankerName]
}

func NewRerankerProvider() (Provider[reranker.Reranker], error) {
	rerankerProviderOnce.Do(func() {
		rerankerProvider = &RerankerProvider[reranker.Reranker]{
			rerankers: make(map[string]reranker.Reranker),
			mux:       &sync.Mutex{},
		}
		e5Reranker, err := reranker.NewE5Reranker()
		if err != nil {
			logrus.Errorf("failed to create e5Reranker client. %v", err)
		}

		rerankerProvider.Add("e5", e5Reranker)
		logrus.Debugf("reranker provider is created. %v", rerankerProvider)
	})

	if err != nil {
		return nil, err
	}
	return rerankerProvider, nil
}
