package searcher

import (
	"context"
	"errors"

	"bigkinds.or.kr/search/service/searcher/opensearch"
	"bigkinds.or.kr/search/service/searcher/opensearch/query_strategy"
	"google.golang.org/protobuf/types/known/structpb"

	"bigkinds.or.kr/pkg/log"
	"bigkinds.or.kr/pkg/search_engine"
	"bigkinds.or.kr/proto"

	"bigkinds.or.kr/proto/searcher"
	ps "bigkinds.or.kr/proto/searcher/service"
)

type OpenSearchSearcher struct {
	QueryStrategyProvider opensearch.Provider[query_strategy.QueryStrategy]
	SearcherConfig        *searcher.SearcherConfig
}

func (s *OpenSearchSearcher) Search(ctx context.Context, req *proto.SearchRequest, config *proto.SearchConfig) ([]*proto.Item, error) {
	// get logger
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("search")
	}

	var filters []*structpb.Struct
	if req.Filter != nil && req.Filters != nil {
		err = errors.New("cannot use `filter` and `filters` at the same time")
		logger.Error(err.Error())
		return nil, err
	}

	if req.Filter != nil {
		filters = []*structpb.Struct{req.Filter}
	} else if req.Filters != nil {
		filters = req.Filters
	}

	// create search request
	searchRequest := &ps.OSSearcherSearchRequest{
		Query:   req.Query,
		Exclude: req.Exclude,
		Size:    req.Size,
		Filters: filters,
	}

	if s.SearcherConfig == nil {
		errMsg := "no opensearch searcher config. Please check searcher config"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// check if searcher config has opensearch config
	if s.SearcherConfig.GetOpensearch() == nil {
		errMsg := "no opensearch config. Please check searcher config"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// get opensearch client
	osClient, err := search_engine.GetOpenSearchClient()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	searcher := &opensearch.OpenSearch{
		Client:                osClient,
		QueryStrategyProvider: s.QueryStrategyProvider,
		Config:                s.SearcherConfig.GetOpensearch(),
	}

	resp, err := searcher.Search(ctx, searchRequest)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return resp.Items, nil
}

func NewOpenSearchSearcher(provider opensearch.Provider[query_strategy.QueryStrategy], config *searcher.SearcherConfig) (*OpenSearchSearcher, error) {
	return &OpenSearchSearcher{
		QueryStrategyProvider: provider,
		SearcherConfig:        config,
	}, nil
}
