package model

type IssueTopicSummary struct {
	Topic_title   string   `json:"topic_title"`
	Topic_content string   `json:"topic_content"`
	News_count    int      `json:"news_count"`
	News_ids      []string `json:"news_ids"`
}
