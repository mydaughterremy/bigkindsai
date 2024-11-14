package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"bigkinds.or.kr/pkg/log"
	"bigkinds.or.kr/proto"
	"bigkinds.or.kr/search/model"
	pa "bigkinds.or.kr/search/service/aggregator"
	"bigkinds.or.kr/search/service/provider"
	pr "bigkinds.or.kr/search/service/reranker"
	ps "bigkinds.or.kr/search/service/searcher"
	"go.uber.org/zap"
)

type SearchServiceServer struct {
	proto.UnimplementedSearchServiceServer
	SearcherProvider provider.Provider[ps.Searcher]
	RerankerProvider provider.Provider[pr.Reranker]
	SearchConfig     *proto.SearchConfig
}

func calculateMS(ns int64) float64 {
	return float64(ns) / 1000000
}

func getSearcher(provider provider.Provider[ps.Searcher], searcher string) (ps.Searcher, error) {
	se := provider.Get(searcher)
	if se == nil {
		return nil, errors.New("no searcher")
	}
	return se, nil
}

func getReranker(provider provider.Provider[pr.Reranker], reranker string) (pr.Reranker, error) {
	r := provider.Get(reranker)
	if r == nil {
		return nil, errors.New("no reranker")
	}
	return r, nil
}

func getRerankQuery(req *proto.SearchRequest, rerankerConfig *proto.Reranker) (string, error) {
	var rerankQuery string
	var ok bool
	switch src := rerankerConfig.QuerySource.Source; src.(type) {
	case *proto.Reranker_QuerySource_RawQuery_:
		rerankQuery = req.RawQuery
	case *proto.Reranker_QuerySource_Query_:
		rerankQuery, ok = req.Query[src.(*proto.Reranker_QuerySource_Query_).Query.Field]
		if !ok {
			return "", errors.New("field from reranker config not found in real query")
		}
	case *proto.Reranker_QuerySource_ConcatQueryAndRawQuery_:
		query, ok := req.Query[src.(*proto.Reranker_QuerySource_ConcatQueryAndRawQuery_).ConcatQueryAndRawQuery.Field]
		if !ok {
			return "", errors.New("field from reranker config not found in real query")
		}
		rerankQuery = query + " " + req.RawQuery
	default:
		return "", errors.New("invalid query source")
	}
	return rerankQuery, nil
}

func (s *SearchServiceServer) getSearchResults(ctx context.Context, searcher ps.Searcher, req *proto.SearchRequest, searchConfig *proto.SearchConfig, profile *model.SearchProfile) ([]*proto.Item, error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("search")
	}

	// consider reranker
	originalTopk := req.Size
	if searchConfig.Reranker != nil {
		req.Size = req.Size * 20 // if reranker is enabled, search 20x more items
	}

	searcherStartTime := time.Now()
	items, err := searcher.Search(ctx, req, searchConfig)
	if err != nil {
		return nil, err
	}
	profile.SearcherTime = calculateMS(time.Since(searcherStartTime).Nanoseconds())

	// rerank

	if rerankerConfig := searchConfig.Reranker; rerankerConfig != nil {
		rerankerStartTime := time.Now()
		reranker, err := getReranker(s.RerankerProvider, rerankerConfig.RerankerType)
		if err != nil {
			return nil, err
		}

		// create rerank query
		rerankQuery, err := getRerankQuery(req, rerankerConfig)
		if err != nil {
			return nil, err
		}

		rerankedItems, err := reranker.Rerank(ctx, rerankerConfig, rerankQuery, items, originalTopk)
		if err == pr.ErrRerankerTimeout {
			logger.Error(err.Error())
			rerankedItems = items[:originalTopk]
		} else if err != nil {
			return nil, err
		}
		profile.RerankerTime = calculateMS(time.Since(rerankerStartTime).Nanoseconds())
		return rerankedItems, nil
	} else {
		return items, nil
	}
}

func (s *SearchServiceServer) Search(ctx context.Context, req *proto.SearchRequest) (*proto.SearchResponse, error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("search")
	}

	// create search service profile
	profile := &model.SearchServiceProfile{}

	// set default size
	if req.Size == 0 {
		req.Size = 10
	}

	if s.SearchConfig == nil {
		errMsg := "no search config"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	searcher, err := getSearcher(s.SearcherProvider, s.SearchConfig.GetSearcherType())
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	// search
	searchStartTime := time.Now()
	profile.SearchProfile = &model.SearchProfile{}
	items, err := s.getSearchResults(ctx, searcher, req, s.SearchConfig, profile.SearchProfile)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	profile.SearchTime = calculateMS(time.Since(searchStartTime).Nanoseconds())
	profile.TotalElapsedTime = calculateMS(time.Since(searchStartTime).Nanoseconds())
	logger.Info("search profile", zap.Any("profile", profile))
	return &proto.SearchResponse{
		Items: items,
	}, nil
}

func (s *SearchServiceServer) aggregateMsearchResults(ctx context.Context, req *proto.MSearchRequest, searchConfig *proto.SearchConfig, itemsList []*proto.Items) ([]*proto.Items, error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("search")
	}

	if req.Aggregate == nil { // if aggregate is not set, return itemsList
		return itemsList, nil
	}

	// aggregate items
	var rerankedItems []*proto.Item
	switch r := strings.ToLower(req.Aggregate.Method); r {
	case "flatten_reranker":
		var rerankerType string
		var useReranker bool
		if rerankerConfig := searchConfig.Reranker; rerankerConfig != nil {
			rerankerType = rerankerConfig.RerankerType
			for _, r := range req.Requests {
				rerankQuery, err := getRerankQuery(r, rerankerConfig)
				if err != nil {
					return nil, err
				}
				if rerankQuery != req.RawQuery {
					useReranker = true
					break
				}
			}
		} else {
			useReranker = true
			logger.Info("reranker config is not set. use default reranker(e5)")
			rerankerType = "e5" // default reranker
		}

		if useReranker {
			reranker, err := getReranker(s.RerankerProvider, rerankerType)
			if err != nil {
				return nil, err
			}

			// create aggregator
			aggregator := &pa.FlattenReranker{
				Reranker: reranker,
			}

			// In msearch, rerank query is same for raw query
			if req.RawQuery == "" {
				return nil, errors.New("raw query is empty. reranker requires raw query in msearch")
			}

			// create rerankRequest
			rerankedItems, err = aggregator.Aggregate(ctx, &proto.Reranker{}, req.RawQuery, itemsList, req.Size)
			if err == pr.ErrRerankerTimeout {
				// use rrf to fallback
				logger.Warn("reranker timeout. fallback to rrf")
				aggregator := &pa.RRF{}
				rerankedItems, err = aggregator.Aggregate(ctx, &proto.Reranker{}, "", itemsList, req.Size)
				if err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			}
		} else {
			// sort items by score
			for _, items := range itemsList {
				rerankedItems = append(rerankedItems, items.Items...)
			}
			sort.Slice(rerankedItems, func(i, j int) bool {
				return rerankedItems[i].Score > rerankedItems[j].Score
			})
		}

	case "rrf":
		aggregator := &pa.RRF{}

		// create rerankRequest
		rerankedItems, err = aggregator.Aggregate(ctx, &proto.Reranker{}, "", itemsList, req.Size)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid aggregate method")
	}

	// if preserve_source, split rerankedItems into itemsList
	aggregatedItemsList := make([]*proto.Items, 0, len(itemsList))
	if req.Aggregate.PreserveSource {
		for i, items := range itemsList {
			aggregatedItems := make([]*proto.Item, 0, req.Size)
			for _, item := range items.Items {
				for _, r := range rerankedItems {
					if item.Id == r.Id {
						aggregatedItems = append(aggregatedItems, item)
					}
				}
			}
			aggregatedItemsList = append(aggregatedItemsList, &proto.Items{
				Id:    req.Requests[i].Id,
				Items: aggregatedItems,
			})
		}
	} else {
		aggregatedItemsList = append(aggregatedItemsList, &proto.Items{
			Items: rerankedItems,
		})
	}

	return aggregatedItemsList, nil
}

func (s *SearchServiceServer) getMultiSearchResults(ctx context.Context, searcher ps.Searcher, req *proto.MSearchRequest, searchConfig *proto.SearchConfig, profile *model.MSearchProfile) ([]*proto.Items, error) {
	// when reranker is set and query source is raw query, set raw query if not exists
	if rerankerConfig := searchConfig.Reranker; rerankerConfig != nil {
		switch src := rerankerConfig.QuerySource.Source; src.(type) {
		case *proto.Reranker_QuerySource_RawQuery_:
			if req.RawQuery == "" {
				return nil, errors.New("raw query is empty. reranker requires raw query in msearch if reranker is set in search config")
			}
			for _, r := range req.Requests {
				if r.RawQuery == "" {
					r.RawQuery = req.RawQuery
				}
			}
		}
	}

	// create search service profile
	profile.SearchProfile = make([]*model.SearchProfile, len(req.Requests))

	// using go routine, msearch
	aggregatedItemsList := make([]*proto.Items, 0, len(req.Requests))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 1) // 에러를 전달할 채널 생성

	for i, r := range req.Requests {
		wg.Add(1)
		go func(index int, rCopy *proto.SearchRequest) {
			defer wg.Done()
			profile.SearchProfile[index] = &model.SearchProfile{}
			if res, err := s.getSearchResults(ctx, searcher, rCopy, searchConfig, profile.SearchProfile[index]); err != nil {
				errChan <- err
				cancel()
			} else {
				mu.Lock()
				aggregatedItemsList = append(aggregatedItemsList, &proto.Items{
					Id:    rCopy.Id,
					Items: res,
				})
				mu.Unlock()
			}
		}(i, r)
	}

	wg.Wait()
	close(errChan)

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		break
	}
	return aggregatedItemsList, nil
}

func (s *SearchServiceServer) MSearch(ctx context.Context, req *proto.MSearchRequest) (*proto.MSearchResponse, error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("search")
	}

	// create search service profile
	profile := &model.SearchServiceProfile{}

	if s.SearchConfig == nil {
		errMsg := "no search config"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// check searcher type
	if searcherType := s.SearchConfig.GetSearcherType(); searcherType != "opensearch" {
		errMsg := "only opensearch searcher type is supported for msearch"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// get searcher
	searcher, err := getSearcher(s.SearcherProvider, s.SearchConfig.GetSearcherType())
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	// msearch
	msearchStartTime := time.Now()
	profile.MSearchProfile = &model.MSearchProfile{}
	itemsList, err := s.getMultiSearchResults(ctx, searcher, req, s.SearchConfig, profile.MSearchProfile)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	profile.SearchTime = calculateMS(time.Since(msearchStartTime).Nanoseconds())

	aggregateStartTime := time.Now()
	// aggregate
	itemsList, err = s.aggregateMsearchResults(ctx, req, s.SearchConfig, itemsList)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	profile.MSearchProfile.AggregatorTime = calculateMS(time.Since(aggregateStartTime).Nanoseconds())
	profile.MSearchProfile.TotalElapsedTime = calculateMS(time.Since(msearchStartTime).Nanoseconds())

	logger.Info("msearch profile", zap.Any("profile", profile))
	return &proto.MSearchResponse{
		Items: itemsList,
	}, nil
}
