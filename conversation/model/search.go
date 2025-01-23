package model

type PublishedDateRange struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type Items struct {
	Items []*Reference `json:"items"`
}

type ItemsWithId struct {
	Id    string       `json:"id"`
	Items []*Reference `json:"items"`
}

type ItemsWithIds struct {
	Items []*ItemsWithId `json:"items"`
}

type MSearchBody struct {
	Requests  []*SearchRequest  `json:"requests"`
	Size      int               `json:"size"`
	RawQuery  string            `json:"raw_query"`
	Aggregate *MSearchAggregate `json:"aggregate"`
}

type MSearchAggregate struct {
	Method         string `json:"method"`
	PreserveSource bool   `json:"preserve_source"`
}

type SearchRequest struct {
	Id       string                   `json:"id"`
	Query    map[string]string        `json:"query"`
	Filters  []map[string]interface{} `json:"filters"`
	Size     int                      `json:"size"`
	MinScore float64                  `json:"min_score"`
}
