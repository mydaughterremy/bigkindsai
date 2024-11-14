package function

import (
	"errors"
	"testing"
	"time"

	"bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func getMockSearchPlugin() *SearchPlugin {
	mockTime, _ := time.Parse(time.RFC3339, "2023-05-11T12:00:00.000+09:00")
	return &SearchPlugin{
		currentTime: utils.CurrentTime{
			Time: mockTime,
		},
	}
}

func TestGetReference(t *testing.T) {
	testCases := []struct {
		name         string
		itemsWithIds []*model.ItemsWithId
		expected     []*model.Reference
		err          error
	}{
		{
			name:         "len(itemsWithIds) == 0",
			itemsWithIds: []*model.ItemsWithId{},
			expected:     []*model.Reference{},
			err:          nil,
		},
		{
			name: "len(itemsWithIds) == 1",
			itemsWithIds: []*model.ItemsWithId{
				{
					Id: "1",
					Items: []*model.Reference{
						{
							ID: "ref1",
						},
					},
				},
			},
			expected: []*model.Reference{
				{
					ID: "ref1",
				},
			},
			err: nil,
		},
		{
			name: "len(itemsWithIds) == 2",
			itemsWithIds: []*model.ItemsWithId{
				{
					Id: "1",
					Items: []*model.Reference{
						{
							ID: "ref1",
						},
					},
				},
				{
					Id: "2",
					Items: []*model.Reference{
						{
							ID: "ref2",
						},
					},
				},
			},
			expected: nil,
			err:      errors.New("too many results"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getReference(tc.itemsWithIds)
			assert.Equal(t, tc.expected, actual)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestHasFutureDate(t *testing.T) {
	testCases := []struct {
		name               string
		publishedDateRange *model.PublishedDateRange
		expected           bool
		err                error
	}{
		{
			name: "start date is empty and end date is not future date",
			publishedDateRange: &model.PublishedDateRange{
				EndDate: "2023-04-23T12:00:00.000+09:00",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "start date is empty and end date is future date",
			publishedDateRange: &model.PublishedDateRange{
				EndDate: "2023-05-23T12:00:00.000+09:00",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "start date is not empty and end date is future date",
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-04-23T12:00:00.000+09:00",
				EndDate:   "2023-05-23T12:00:00.000+09:00",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "start date is not empty and end date is not future date",
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-04-23T12:00:00.000+09:00",
				EndDate:   "2023-04-23T12:00:00.000+09:00",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "start date is future date and end date is empty",
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-05-23T12:00:00.000+09:00",
			},
			expected: true,
			err:      nil,
		},
		{
			name: "start date is not future date and end date is empty",
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-04-23T12:00:00.000+09:00",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "start date is future date and end date is future date",
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-05-23T12:00:00.000+09:00",
				EndDate:   "2023-05-24T12:00:00.000+09:00",
			},
			expected: true,
			err:      nil,
		},
	}
	mockSearchPlugin := getMockSearchPlugin()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := mockSearchPlugin.hasFutureDate(tc.publishedDateRange)
			assert.Equal(t, tc.expected, actual)
			assert.Equal(t, tc.err, err)
		})
	}

}

func TestHandleDateRangeNotSpecified(t *testing.T) {
	testCases := []struct {
		name            string
		rawQuery        string
		standaloneQuery string
		topK            int
		expected        *model.MSearchBody
		err             error
	}{
		{
			name:            "valid query",
			rawQuery:        "test-raw-query",
			standaloneQuery: "test-standalone-query",
			topK:            10,
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 10,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"gte": "now-3M",
										"lte": "now",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 10,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"lt": "now-3M",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
				},
				Size:     10,
				RawQuery: "test-raw-query",
				Aggregate: &model.MSearchAggregate{
					Method:         "flatten_reranker",
					PreserveSource: false,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FILTER_ARTICLE_LENGTH", "true")

			actual := handleDateRangeNotSpecified(tc.rawQuery, tc.standaloneQuery, tc.topK)
			assert.Equal(t, tc.expected, actual)
		})
	}

}

func TestHandleDateRangeSpecified(t *testing.T) {
	testCases := []struct {
		name               string
		rawQuery           string
		standaloneQuery    string
		topK               int
		publishedDateRange *model.PublishedDateRange
		expected           *model.MSearchBody
		errExists          bool
	}{
		{
			name:            "valid query: end date is same as current date",
			rawQuery:        "test-raw-query",
			standaloneQuery: "test-standalone-query",
			topK:            10,
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-04-23T12:00:00.000+09:00",
				EndDate:   "2023-05-23T23:59:59.000+09:00",
			},
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 10,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
							{
								"range": map[string]interface{}{
									"published_at": map[string]interface{}{
										"gte": "2023-04-23T12:00:00.000+09:00",
										"lte": "2023-05-23T23:59:59.000+09:00",
									},
								},
							},
						},
					},
				},
				Size:     10,
				RawQuery: "test-raw-query",
			},
			errExists: false,
		},
		{
			name:            "valid query: end date is past date",
			rawQuery:        "test-raw-query",
			standaloneQuery: "test-standalone-query",
			topK:            10,
			publishedDateRange: &model.PublishedDateRange{
				StartDate: "2023-04-23T12:00:00.000+09:00",
				EndDate:   "2023-05-11T23:59:59.000+09:00",
			},
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 10,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
							{
								"range": map[string]interface{}{
									"published_at": map[string]interface{}{
										"gte": "2023-04-23T12:00:00.000+09:00",
										"lte": "2023-05-11T23:59:59.000+09:00",
									},
								},
							},
						},
					},
				},
				Size:     10,
				RawQuery: "test-raw-query",
			},
			errExists: false,
		},
		{
			name:            "invalid query: time format is invalid",
			rawQuery:        "test-raw-query",
			standaloneQuery: "test-standalone-query",
			topK:            10,
			publishedDateRange: &model.PublishedDateRange{
				EndDate: "2023-04-21TST12:00:00.000+09:00",
			},
			expected:  nil,
			errExists: true,
		},
	}
	mockSearchPlugin := getMockSearchPlugin()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FILTER_ARTICLE_LENGTH", "true")

			actual, err := mockSearchPlugin.handleDateRangeSpecified(tc.rawQuery, tc.standaloneQuery, tc.topK, tc.publishedDateRange)
			assert.Equal(t, tc.expected, actual)
			if tc.errExists {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestCreateSearchRequests(t *testing.T) {
	testCases := []struct {
		name      string
		arguments map[string]interface{}
		rawQuery  string
		expected  *model.MSearchBody
	}{
		{
			name: "valid query: date range is not specified (case 1)",
			arguments: map[string]interface{}{
				"standalone_query": "test-standalone-query",
			},
			rawQuery: "test-raw-query",
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 15,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"gte": "now-3M",
										"lte": "now",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 15,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"lt": "now-3M",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
				},
				Size:     15,
				RawQuery: "test-raw-query",
				Aggregate: &model.MSearchAggregate{
					Method:         "flatten_reranker",
					PreserveSource: false,
				},
			},
		},
		{
			name: "valid query: date range is specified and not future (case 2)",
			arguments: map[string]interface{}{
				"standalone_query": "test-standalone-query",
				"published_date_range": map[string]interface{}{
					"start_date": "2023-04-23T12:00:00.000+09:00",
					"end_date":   "2023-05-03T23:59:59.000+09:00",
				},
			},
			rawQuery: "test-raw-query",
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 15,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
							{
								"range": map[string]interface{}{
									"published_at": map[string]interface{}{
										"gte": "2023-04-23T12:00:00.000+09:00",
										"lte": "2023-05-03T23:59:59.000+09:00",
									},
								},
							},
						},
					},
				},
				Size:     15,
				RawQuery: "test-raw-query",
			},
		},
		{
			name: "valid query: date range is specified and future(case 3)",
			arguments: map[string]interface{}{
				"standalone_query": "test-standalone-query",
				"published_date_range": map[string]interface{}{
					"start_date": "2023-05-28T12:00:00.000+09:00",
				},
			},
			rawQuery: "test-raw-query",
			expected: &model.MSearchBody{
				Requests: []*model.SearchRequest{
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 15,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"gte": "now-3M",
										"lte": "now",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
					{
						Query: map[string]string{
							"title":   "test-standalone-query",
							"content": "test-standalone-query",
						},
						Size: 15,
						Filters: []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"published_at": map[string]string{
										"lt": "now-3M",
									},
								},
							},
							{
								"range": map[string]interface{}{
									"article_length": map[string]interface{}{
										"gt": 200,
									},
								},
							},
						},
					},
				},
				Size:     15,
				RawQuery: "test-raw-query",
				Aggregate: &model.MSearchAggregate{
					Method:         "flatten_reranker",
					PreserveSource: false,
				},
			},
		},
	}

	mockSearchPlugin := getMockSearchPlugin()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extraArgs := &ExtraArgs{
				RawQuery: tc.rawQuery,
			}
			t.Setenv("FILTER_ARTICLE_LENGTH", "true")

			actual, err := mockSearchPlugin.createSearchRequests(tc.arguments, extraArgs)
			assert.Equal(t, tc.expected, actual)
			assert.Nil(t, err)
		})
	}

}
