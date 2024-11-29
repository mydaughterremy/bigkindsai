package function

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	model "bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/pkg/utils"
)

var ErrSearchFunctionFailed = errors.New("search function failed")

type SearchPlugin struct {
	CurrentTime utils.CurrentTime
}

func mergeChunks(chunks []*model.Reference) []*model.Reference {
	const maxChunkSize = 1000
	const maxChunkNumber = 5

	mergedChunks := make([]*model.Reference, 0, maxChunkNumber)

	for _, chunk := range chunks {
		merged := false
		for i, mergedChunk := range mergedChunks {
			if mergedChunk.Attributes.NewsID == chunk.Attributes.NewsID {
				mergedChunks[i].Attributes.Content += "\n" + chunk.Attributes.Content
				chunkSize := min(len(mergedChunks[i].Attributes.Content), maxChunkSize)
				mergedChunks[i].Attributes.Content = mergedChunks[i].Attributes.Content[:chunkSize]
				merged = true
				break
			}
		}
		if !merged {
			mergedChunks = append(mergedChunks, chunk)
		}
		if len(mergedChunks) >= maxChunkNumber {
			break
		}
	}

	return mergedChunks
}

func getReference(ItemsWithId []*model.ItemsWithId) ([]*model.Reference, error) {
	if len(ItemsWithId) == 0 {
		slog.Info("No search results")
		return []*model.Reference{}, nil
	} else if len(ItemsWithId) == 1 {
		return ItemsWithId[0].Items, nil
	} else {
		return nil, errors.New("too many results")
	}
}

func unmarshalItemsWithIds(result []byte) ([]*model.ItemsWithId, error) {
	var items model.ItemsWithIds
	err := json.Unmarshal(result, &items)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling references: %s", err.Error())
	}
	return items.Items, nil
}

func marshalReference(references []*model.Reference) ([]byte, error) {
	items := model.Items{
		Items: references,
	}
	b, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("error marshalling references: %s", err.Error())
	}

	return b, nil
}

func (s *SearchPlugin) hasFutureDate(dateRange *model.PublishedDateRange) (bool, error) {
	if dateRange.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, dateRange.StartDate)
		if err != nil {
			slog.Info("The date in `published_date_range` start date is invalid")
			return false, err
		}
		return startTime.After(s.CurrentTime.Time), nil
	}
	return false, nil
}

func handleDateRangeNotSpecified(rawQuery, standaloneQuery string, topK int) *model.MSearchBody {
	filtersRecent3M := []map[string]interface{}{
		{
			"range": map[string]interface{}{
				"published_at": map[string]string{
					"gte": "now-3M",
					"lte": "now",
				},
			},
		},
	}
	filtersBefore3M := []map[string]interface{}{
		{
			"range": map[string]interface{}{
				"published_at": map[string]string{
					"lt": "now-3M",
				},
			},
		},
	}
	if os.Getenv("FILTER_ARTICLE_LENGTH") == "true" {
		filterArticleLength := map[string]interface{}{
			"range": map[string]interface{}{"article_length": map[string]interface{}{"gt": 200}},
		}
		filtersRecent3M = append(filtersRecent3M, filterArticleLength)
		filtersBefore3M = append(filtersBefore3M, filterArticleLength)
	}
	return &model.MSearchBody{
		Requests: []*model.SearchRequest{
			{
				Query: map[string]string{
					"title":   standaloneQuery,
					"content": standaloneQuery,
				},
				Size:    topK,
				Filters: filtersRecent3M,
			},
			{
				Query: map[string]string{
					"title":   standaloneQuery,
					"content": standaloneQuery,
				},
				Size:    topK,
				Filters: filtersBefore3M,
			},
		},
		Size:     topK,
		RawQuery: rawQuery,
		Aggregate: &model.MSearchAggregate{
			Method:         "rrf",
			PreserveSource: false,
		},
	}
}

func (s *SearchPlugin) handleDateRangeSpecified(rawQuery, standaloneQuery string, topK int, publishedDateRange *model.PublishedDateRange) (*model.MSearchBody, error) {
	body := &model.MSearchBody{
		Size:     topK,
		RawQuery: rawQuery,
	}

	baseQuery := map[string]string{
		"title":   standaloneQuery,
		"content": standaloneQuery,
	}

	// filter for published date range
	publishedDateRangeFilter := make(map[string]interface{}, 0)
	if publishedDateRange.StartDate != "" {
		_, err := time.Parse(time.RFC3339, publishedDateRange.StartDate)
		if err != nil {
			slog.Info("The date in `published_date_range` start date is invalid")
			return nil, err
		}
		publishedDateRangeFilter["gte"] = publishedDateRange.StartDate
	}
	if publishedDateRange.EndDate != "" {
		_, err := time.Parse(time.RFC3339, publishedDateRange.EndDate)
		if err != nil {
			slog.Info("The date in `published_date_range` end date is invalid")
			return nil, err
		}
		publishedDateRangeFilter["lte"] = publishedDateRange.EndDate
	}

	var searchRequestFilters []map[string]interface{}

	if os.Getenv("FILTER_ARTICLE_LENGTH") == "true" {
		searchRequestFilters = append(searchRequestFilters, map[string]interface{}{
			"range": map[string]interface{}{"article_length": map[string]interface{}{"gt": 200}},
		})
	}
	if len(publishedDateRangeFilter) > 0 {
		searchRequestFilters = append(searchRequestFilters, map[string]interface{}{
			"range": map[string]interface{}{"published_at": publishedDateRangeFilter},
		})
	}
	body.Requests = append(body.Requests, &model.SearchRequest{
		Query:   baseQuery,
		Size:    topK,
		Filters: searchRequestFilters,
	})

	return body, nil
}

func (s *SearchPlugin) createSearchRequests(arguments map[string]interface{}, extraArgs *ExtraArgs) (*model.MSearchBody, error) {
	var body *model.MSearchBody
	const topK = 15
	standaloneQuery, ok := arguments["standalone_query"].(string)
	rawQuery := extraArgs.RawQuery
	if !ok {
		standaloneQuery = rawQuery // fallback to raw query
	}

	_, ok = arguments["published_date_range"]
	if !ok {
		// case 1: no date range specified
		body = handleDateRangeNotSpecified(rawQuery, standaloneQuery, topK)
	} else {
		// case 2: date range specified
		var publishedDateRange model.PublishedDateRange
		jsonData, err := json.Marshal(arguments["published_date_range"])
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonData, &publishedDateRange)
		if err != nil {
			return nil, err
		}

		ok, err = s.hasFutureDate(&publishedDateRange)
		if err != nil || ok {
			// case 2-2: date range is future -> fallback to case 1
			body = handleDateRangeNotSpecified(rawQuery, standaloneQuery, topK)
		} else {
			// case 2-1: date range is not future
			body, err = s.handleDateRangeSpecified(rawQuery, standaloneQuery, topK, &publishedDateRange)
			if err != nil {
				return nil, err
			}
		}
	}

	provider := extraArgs.Provider
	if len(provider) > 0 {
		for _, request := range body.Requests {
			request.Query["provider"] = provider
		}
	}

	return body, nil
}

func (s *SearchPlugin) Call(ctx context.Context, arguments map[string]interface{}, extraArgs *ExtraArgs) ([]byte, error) {
	endpoint := os.Getenv("UPSTAGE_SEARCHSERVICE_MSEARCH_ENDPOINT")
	if endpoint == "" {
		return nil, errors.New("search service endpoint not set")
	}

	query, err := s.createSearchRequests(arguments, extraArgs)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrSearchFunctionFailed
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	itemsWithIds, err := unmarshalItemsWithIds(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling references: %s", err.Error())
	}
	references, err := getReference(itemsWithIds)
	if err != nil {
		return nil, err
	}

	merged := mergeChunks(references)
	body, err = marshalReference(merged)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (s *SearchPlugin) Definition() model.Function {
	firstExampleStartTime := time.Date(s.CurrentTime.Time.Year(), s.CurrentTime.Time.Month(), s.CurrentTime.Time.Day(), 0, 0, 0, 0, s.CurrentTime.Location).AddDate(0, 0, -30).Format("2006-01-02T15:04:05-07:00")
	firstExampleEndTime := time.Date(s.CurrentTime.Time.Year(), s.CurrentTime.Time.Month(), s.CurrentTime.Time.Day(), 23, 59, 59, 999999, s.CurrentTime.Location).Format("2006-01-02T15:04:05-07:00")
	secondExampleStartTime := time.Date(2011, 1, 1, 0, 0, 0, 0, s.CurrentTime.Location).Format("2006-01-02T15:04:05-07:00")
	secondExampleEndTime := time.Date(2016, 12, 31, 23, 59, 59, 999999, s.CurrentTime.Location).Format("2006-01-02T15:04:05-07:00")
	return model.Function{
		Name: "search",
		Description: fmt.Sprintf(`Use this function to get external knowledge to answer information questions. e.g. "~가 누구야?", "~가 뭐야?", "~에 대해 알려줘", "~는 무엇인가요?".

<example>
question: "최근 애플 근황 뉴스"
{{
	"standalone_query": "애플",
	"published_date_range": {{
	"start_date": "%s",
	"end_date": "%s"
	}}
}}
</example>
When the user's question pertains to statistics, and the data is typically published after the end of the year, you should set the 'start_date' to the next year's first day and 'end_date' to the current year's last day.
    <example>
    question: "2010년부터 2015년 출산율 알려줘"
    {{
    "standalone_query": "출산율",
    "published_date_range": {{
    "start_date": "%s",
    "end_date": "%s"
    }}
    }}
    </example>
If the user's query does not specify a time period explicitly, you can leave the 'publish_date_range' field empty.
`, firstExampleStartTime, firstExampleEndTime, secondExampleStartTime, secondExampleEndTime),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"standalone_query": map[string]interface{}{
					"type":        "string",
					"description": "The comprehensive query that considers the context of the conversation history.",
				},
				"published_date_range": map[string]interface{}{
					"type": "object",
					"description": `This field defines the time frame for filtering articles based on their publication dates. 
					Extract time-related expressions from the user's query to determine the appropriate date range.
					The term "recent" implies a date range extending back one month from today's date.
					This field should be filled based on the user's query and has no default value.`,
					"properties": map[string]interface{}{
						"start_date": map[string]string{
							"type":        "string",
							"description": "The start date or minimum publication date for articles to be searched.",
						},
						"end_date": map[string]string{
							"type":        "string",
							"description": "The end date or maximum publication date for articles to be searched.",
						},
					},
				},
			},
			"required": []string{"standalone_query"},
		},
	}
}
