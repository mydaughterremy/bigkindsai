package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"bigkinds.or.kr/pkg/log"
	"bigkinds.or.kr/proto"
	"bigkinds.or.kr/proto/searcher"
	"bigkinds.or.kr/proto/searcher/service"
	"bigkinds.or.kr/search/service/searcher/opensearch/compound_strategy"
	q "bigkinds.or.kr/search/service/searcher/opensearch/query_strategy"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"google.golang.org/protobuf/types/known/structpb"
)

type OpenSearch struct {
	Client                *opensearch.Client
	Config                *searcher.OpenSearchSearcher
	QueryStrategyProvider Provider[q.QueryStrategy]
}

func osSearchResponseToProto(resp *opensearchapi.Response) (*service.OSSearcherSearchResponse, error) {
	var r map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("Error parsing the response body: %s", err)
	}

	hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
	items := make([]*proto.Item, len(hits))
	for i, hit := range hits {
		// set id
		id := hit.(map[string]interface{})["_id"].(string)

		// set attributes
		attributes, err := structpb.NewStruct(
			hit.(map[string]interface{})["_source"].(map[string]interface{}),
		)
		if err != nil {
			return nil, err
		}
		items[i] = &proto.Item{
			Id:         id,
			Attributes: attributes,
		}
	}

	return &service.OSSearcherSearchResponse{
		Items: items,
	}, nil
}

func newErrorFromOpenSearchResponse(resp *opensearchapi.Response) error {
	var e map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return fmt.Errorf("Error parsing the response body: %s", err)
	} else {
		return fmt.Errorf("[%s] %s: %s",
			resp.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"],
		)
	}
}

func (s *OpenSearch) createQuery(ctx context.Context, queryStrategy q.QueryStrategy, queryIngredients *q.QueryIngredients) (map[string]interface{}, error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("os-searcher")
	}

	query, err := queryStrategy.CreateOpenSearchQuery(queryIngredients, s.Config)
	if err != nil {
		return nil, fmt.Errorf("Error creating opensearch query: %s", err)
	}

	// if filter is not empty, add filter to query
	if queryIngredients.Filters != nil {
		for _, vs := range queryIngredients.Filters {
			for k, v := range vs.AsMap() {
				switch k {
				case "range":
					if must, ok := query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"]; ok {
						// if query already has the 'must' key, add range filter to 'must'
						must = append(must.([]map[string]interface{}), map[string]interface{}{k: v})
						query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must
					} else {
						// if query does not have the 'must' key, create 'must' key and add range filter to 'must'
						query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = []map[string]interface{}{
							{
								k: v,
							},
						}
					}
				}
			}
		}
	}

	if s.Config.CompoundStrategy != nil {
		switch c := s.Config.CompoundStrategy; c.CompoundStrategy.(type) {
		case *searcher.OpenSearchSearcher_CompoundStrategy_FunctionScore_:
			compoundStrategy := compound_strategy.NewFunctionScoreCompoundStrategy()
			compoundQuery, err := compoundStrategy.CreateCompoundQuery(query, c)
			if err != nil {
				return nil, fmt.Errorf("Error creating compound query: %s", err)
			}
			query = compoundQuery
		default:
			return nil, fmt.Errorf("invalid compound strategy type")
		}
	}

	// set size
	size := int(queryIngredients.Size)
	if size == 0 {
		size = 10
		logger.Info("no size parameter, set size to 10")
	}
	query["size"] = size
	fmt.Printf("%+v\n", query)
	return query, nil
}

func (s *OpenSearch) newOSSearchRequest(ctx context.Context, req *service.OSSearcherSearchRequest) ([]func(*opensearchapi.SearchRequest), error) {
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("os-searcher")
	}

	// create query
	queryStrategyType, err := getQueryStrategyType(s.Config)
	if err != nil {
		return nil, err
	}
	queryStrategy := s.QueryStrategyProvider.Get(queryStrategyType)
	query, err := s.createQuery(ctx, queryStrategy, &q.QueryIngredients{
		Query:   req.Query,
		Exclude: req.Exclude,
		Size:    req.Size,
		Filters: req.Filters,
	})
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("Error encoding query: %s", err)
	}

	index := s.Config.IndexName

	os := s.Client
	searchRequest := []func(*opensearchapi.SearchRequest){
		os.Search.WithContext(ctx),
		os.Search.WithIndex(index),
		os.Search.WithBody(&buf),
		os.Search.WithSort("_score"),
	}

	return searchRequest, nil
}

func getQueryStrategyType(searcherConfig *searcher.OpenSearchSearcher) (string, error) {
	switch searcherConfig.QueryStrategy.(type) {
	case *searcher.OpenSearchSearcher_TextQueryStrategy_:
		return "text", nil
	case *searcher.OpenSearchSearcher_VectorQueryStrategy_:
		return "vector", nil
	default:
		return "", fmt.Errorf("invalid query strategy type")
	}
}

func (s *OpenSearch) Search(ctx context.Context, req *service.OSSearcherSearchRequest) (*service.OSSearcherSearchResponse, error) {
	os := s.Client

	openSearchRequest, err := s.newOSSearchRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	res, err := os.Search(openSearchRequest...)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, newErrorFromOpenSearchResponse(res)
	}

	return osSearchResponseToProto(res)
}
