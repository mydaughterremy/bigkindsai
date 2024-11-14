package model

type Statistics struct {
	UserTrends         UserTrend          `json:"user_trends"`
	UserTrendsHourly   UserTrendHourly    `json:"user_trends_hourly"`
	SatisfactionSurvey SatisfactionSurvey `json:"satisfaction_survey"`
	AskHistory         AskHistory         `json:"ask_history"`
	LLMTokenHistory    LLMTokenHistory    `json:"llm_token_history"`
	KeywordTrends      KeywordTrends      `json:"keyword_trends"`
}

type JobGroupMap map[string]map[string]bool

type UserTrend struct {
	Labels []string        `json:"labels"`
	Data   []UserTrendUnit `json:"data"`
}

type UserTrendUnit struct {
	Label string `json:"label"`
	Data  []int  `json:"data"`
}

type UserTrendHourly struct {
	Labels []string              `json:"labels"`
	Data   []UserTrendHourlyUnit `json:"data"`
}

type UserTrendHourlyUnit struct {
	Label string `json:"label"`
	Data  []int  `json:"data"`
}

type SatisfactionSurvey struct {
	Good float64 `json:"good"`
	Bad  float64 `json:"bad"`
}

type AskHistory struct {
	Labels []string `json:"labels"`
	Data   []int    `json:"data"`
}

type LLMTokenHistory struct {
	Labels []string `json:"labels"`
	Data   []int    `json:"data"`
}

type KeywordSummaryRanking struct {
	From    string                `json:"from"`
	To      string                `json:"to"`
	Ranking []KeywordRankingBlock `json:"ranking"`
}

type KeywordTimeBasedRanking struct {
	DateTime string                `json:"datetime"`
	Ranking  []KeywordRankingBlock `json:"ranking"`
}

type KeywordRankingBlock struct {
	No    int    `json:"no"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type KeywordTrends struct {
	Summary   KeywordSummaryRanking     `json:"summary"`
	TimeBased []KeywordTimeBasedRanking `json:"time_based"`
}
