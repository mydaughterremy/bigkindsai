package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
)

type StatisticsService struct {
	Repository *repository.QARepository
}

const UnitDaily = time.Duration(24) * time.Hour
const DateOutputFormat = "2006-01-02"
const YearMonthOutputFormat = "2006-01"

const (
	PeriodUnitDay   = "DAY"
	PeriodUnitMonth = "MONTH"

	LabelJobGroupETC = "기타"
)

func newJobGroup(jobGroups []string) model.JobGroupMap {
	jobGroupMap := make(model.JobGroupMap)

	for _, jobGroup := range jobGroups {
		jobGroupMap[jobGroup] = map[string]bool{}
	}
	return jobGroupMap
}

func (s StatisticsService) getJobGroupListFromQAs(qas []*model.QA) []string {
	jobGroupMap := map[string]bool{}
	for _, qa := range qas {
		if _, ok := jobGroupMap[qa.JobGroup]; !ok {
			jobGroup := qa.JobGroup
			if len(jobGroup) == 0 {
				jobGroup = LabelJobGroupETC
			}
			jobGroupMap[jobGroup] = true
		}
	}
	var jobGroups []string
	for jobGroup := range jobGroupMap {
		jobGroups = append(jobGroups, jobGroup)
	}
	return jobGroups
}

func (s StatisticsService) compareFunctionForJobGroupSorting(firstJobGroup string, secondJobGroup string) bool {
	if firstJobGroup == LabelJobGroupETC {
		return false
	}
	if secondJobGroup == LabelJobGroupETC {
		return true
	}
	return firstJobGroup < secondJobGroup
}

func (s StatisticsService) extractUserTrends(qas []*model.QA, from *time.Time, to *time.Time, datetimeFormat string, nextFunc func(*time.Time) *time.Time) *model.UserTrend {

	timePtr := from
	datetimeToJobGroup := map[string]model.JobGroupMap{}

	jobGroups := s.getJobGroupListFromQAs(qas)

	var labels []string
	for timePtr.Compare(*to) <= 0 {
		label := timePtr.Format(datetimeFormat)

		labels = append(labels, label)
		datetimeToJobGroup[label] = newJobGroup(jobGroups)

		timePtr = nextFunc(timePtr)
	}

	for _, qa := range qas {
		dateBucket := qa.CreatedAt.Format(datetimeFormat)

		if _, ok := datetimeToJobGroup[dateBucket]; !ok {
			datetimeToJobGroup[dateBucket] = newJobGroup(jobGroups)
		}
		jobGroupMap := datetimeToJobGroup[dateBucket]
		jobGroupLabel := qa.JobGroup
		if len(jobGroupLabel) == 0 {
			jobGroupLabel = LabelJobGroupETC
		}
		jobGroupMap[jobGroupLabel][qa.SessionID] = true
	}

	sortKeys := make([]string, 0, len(datetimeToJobGroup))
	for datetime := range datetimeToJobGroup {
		sortKeys = append(sortKeys, datetime)
	}
	sort.Slice(sortKeys, func(i, j int) bool {
		return sortKeys[i] < sortKeys[j]
	})

	userTrendData := make([]model.UserTrendUnit, len(jobGroups))

	for i := 0; i < len(jobGroups); i += 1 {
		jobGroup := jobGroups[i]
		jobGroupData := make([]int, len(sortKeys))
		for i, key := range sortKeys {
			jobGroupMap := datetimeToJobGroup[key]
			jobGroupData[i] = len(jobGroupMap[jobGroup])
		}
		userTrendData[i] = model.UserTrendUnit{
			Label: jobGroup,
			Data:  jobGroupData,
		}
	}

	sort.Slice(userTrendData, func(i, j int) bool {
		return s.compareFunctionForJobGroupSorting(userTrendData[i].Label, userTrendData[j].Label)
	})

	return &model.UserTrend{
		Labels: labels,
		Data:   userTrendData,
	}
}

func (s StatisticsService) extractUserTrendsHourly(qas []*model.QA) *model.UserTrendHourly {
	hourToJobGroup := map[int]model.JobGroupMap{}

	jobGroups := s.getJobGroupListFromQAs(qas)

	var hourLabels []string
	for hour := 0; hour < 24; hour++ {
		hourLabel := fmt.Sprint(hour)
		hourLabels = append(hourLabels, hourLabel)

		hourToJobGroup[hour] = newJobGroup(jobGroups)
	}

	for _, qa := range qas {
		hour := qa.CreatedAt.Hour()
		jobGroupLabel := qa.JobGroup
		if len(jobGroupLabel) == 0 {
			jobGroupLabel = LabelJobGroupETC
		}
		hourToJobGroup[hour][jobGroupLabel][qa.SessionID] = true
	}

	userTrendHourlyData := make([]model.UserTrendHourlyUnit, len(jobGroups))

	for i := 0; i < len(jobGroups); i += 1 {
		jobGroup := jobGroups[i]
		hourlyData := make([]int, 24)
		for hour, jobGroupMap := range hourToJobGroup {
			hourlyData[hour] = len(jobGroupMap[jobGroup])
		}
		userTrendHourlyData[i] = model.UserTrendHourlyUnit{
			Label: jobGroup,
			Data:  hourlyData,
		}
	}

	sort.Slice(userTrendHourlyData, func(i, j int) bool {
		return s.compareFunctionForJobGroupSorting(userTrendHourlyData[i].Label, userTrendHourlyData[j].Label)
	})

	return &model.UserTrendHourly{
		Labels: hourLabels,
		Data:   userTrendHourlyData,
	}
}

func (s StatisticsService) extractSatisfactionSurvey(qas []*model.QA) *model.SatisfactionSurvey {
	good, bad := 0, 0
	for _, qa := range qas {
		if qa.Vote == "up" {
			good++
		} else if qa.Vote == "down" {
			bad++
		}
	}
	total := good + bad
	goodPercent, badPercent := 0.0, 0.0
	if total > 0 {
		goodPercent = float64(good) / float64(total) * 100
		badPercent = float64(bad) / float64(total) * 100
	} else {
		goodPercent = -1
		badPercent = -1
	}

	return &model.SatisfactionSurvey{
		Good: goodPercent,
		Bad:  badPercent,
	}
}

func (s StatisticsService) extractAskHistoryDaily(qas []*model.QA, from *time.Time, to *time.Time, datetimeFormat string, nextFunc func(*time.Time) *time.Time) *model.AskHistory {
	timePtr := from
	datetimeToCount := map[string]int{}

	var labels []string
	for timePtr.Compare(*to) <= 0 {
		label := timePtr.Format(datetimeFormat)

		labels = append(labels, label)
		datetimeToCount[label] = 0

		timePtr = nextFunc(timePtr)
	}

	for _, qa := range qas {
		dateBucket := qa.CreatedAt.Format(datetimeFormat)
		datetimeToCount[dateBucket]++
	}

	sortKeys := make([]string, 0, len(datetimeToCount))
	for datetime := range datetimeToCount {
		sortKeys = append(sortKeys, datetime)
	}
	sort.Slice(sortKeys, func(i, j int) bool {
		return sortKeys[i] < sortKeys[j]
	})

	askHistoryData := []int{}
	for _, key := range sortKeys {
		askHistoryData = append(askHistoryData, datetimeToCount[key])
	}

	return &model.AskHistory{
		Labels: labels,
		Data:   askHistoryData,
	}
}

func (s StatisticsService) extractLLMTokenHistoryDaily(qas []*model.QA, from *time.Time, to *time.Time, datetimeFormat string, nextFunc func(*time.Time) *time.Time) *model.LLMTokenHistory {
	timePtr := from
	datetimeToCount := map[string]int{}

	var labels []string
	for timePtr.Compare(*to) <= 0 {
		label := timePtr.Format(datetimeFormat)

		labels = append(labels, label)
		datetimeToCount[label] = 0

		timePtr = nextFunc(timePtr)
	}

	for _, qa := range qas {
		dateBucket := qa.CreatedAt.Format(datetimeFormat)
		datetimeToCount[dateBucket] += qa.TokenCount
	}

	sortKeys := make([]string, 0, len(datetimeToCount))
	for datetime := range datetimeToCount {
		sortKeys = append(sortKeys, datetime)
	}
	sort.Slice(sortKeys, func(i, j int) bool {
		return sortKeys[i] < sortKeys[j]
	})

	var llmTokenHistoryData []int
	for _, key := range sortKeys {
		llmTokenHistoryData = append(llmTokenHistoryData, datetimeToCount[key])
	}

	return &model.LLMTokenHistory{
		Labels: labels,
		Data:   llmTokenHistoryData,
	}
}

func (s StatisticsService) extractKeywordTrends(qas []*model.QA, from *time.Time, to *time.Time, datetimeFormat string, nextFunc func(*time.Time) *time.Time) *model.KeywordTrends {
	type KeywordRanking map[string]int

	timePtr := from
	keywordRankingSummary := KeywordRanking{}
	datetimeToKeywordRanking := map[string]KeywordRanking{}

	for timePtr.Compare(*to) <= 0 {
		label := timePtr.Format(datetimeFormat)
		datetimeToKeywordRanking[label] = KeywordRanking{}

		timePtr = nextFunc(timePtr)
	}

	for _, qa := range qas {
		dateBucket := qa.CreatedAt.Format(datetimeFormat)
		if _, ok := datetimeToKeywordRanking[dateBucket]; !ok {
			datetimeToKeywordRanking[dateBucket] = KeywordRanking{}
		}
		dailyRanking := datetimeToKeywordRanking[dateBucket]
		for _, keyword := range qa.Keywords {
			if _, ok := keywordRankingSummary[keyword]; !ok {
				keywordRankingSummary[keyword] = 0
			}
			keywordRankingSummary[keyword] = keywordRankingSummary[keyword] + 1

			if _, ok := dailyRanking[keyword]; !ok {
				dailyRanking[keyword] = 0
			}
			dailyRanking[keyword] = dailyRanking[keyword] + 1
		}
	}

	topN := 10

	var summary model.KeywordSummaryRanking
	summary.From = from.Format(datetimeFormat)
	summary.To = to.Format(datetimeFormat)
	summaryRanking := make([]model.KeywordRankingBlock, len(keywordRankingSummary))
	summaryIndex := 0
	for kw, count := range keywordRankingSummary {
		summaryRanking[summaryIndex] = model.KeywordRankingBlock{
			Label: kw,
			Count: count,
		}
		summaryIndex += 1
	}
	sort.Slice(summaryRanking, func(i, j int) bool {
		return summaryRanking[i].Count > summaryRanking[j].Count
	})
	for i := 0; i < len(summaryRanking); i++ {
		summaryRanking[i].No = i + 1
	}
	summary.Ranking = summaryRanking[:min(topN, len(summaryRanking))]

	daily := make([]model.KeywordTimeBasedRanking, len(datetimeToKeywordRanking))
	dailyIndex := 0
	for datetime, ranking := range datetimeToKeywordRanking {
		dailyRanking := make([]model.KeywordRankingBlock, len(ranking))
		rankingIndex := 0
		for kw, count := range ranking {
			dailyRanking[rankingIndex] = model.KeywordRankingBlock{
				Label: kw,
				Count: count,
			}
			rankingIndex += 1
		}
		sort.Slice(dailyRanking, func(i, j int) bool {
			return dailyRanking[i].Count > dailyRanking[j].Count
		})
		for i := 0; i < len(dailyRanking); i++ {
			dailyRanking[i].No = i + 1
		}

		daily[dailyIndex] = model.KeywordTimeBasedRanking{
			DateTime: datetime,
			Ranking:  dailyRanking,
		}
		dailyIndex += 1
	}

	for i := 0; i < len(daily); i++ {
		if len(daily[i].Ranking) > topN {
			daily[i].Ranking = daily[i].Ranking[:topN]
		}
	}

	return &model.KeywordTrends{
		Summary:   summary,
		TimeBased: daily,
	}
}

func (s StatisticsService) GetStatistics(ctx context.Context, from *time.Time, to *time.Time, unit string) (*model.Statistics, error) {
	qas, err := s.Repository.ListQAs(ctx, from, to)
	if err != nil {
		return &model.Statistics{}, err
	}

	var datetimeFormat string
	var nextFunc func(timePtr *time.Time) *time.Time
	if unit == PeriodUnitMonth {
		datetimeFormat = YearMonthOutputFormat
		nextFunc = func(timePtr *time.Time) *time.Time {
			nextPtr := timePtr.AddDate(0, 1, 0)
			return &nextPtr
		}
	} else { // unit == PeriodUnitDay ("DAY")
		datetimeFormat = DateOutputFormat
		nextFunc = func(timePtr *time.Time) *time.Time {
			nextPtr := timePtr.Add(UnitDaily)
			return &nextPtr
		}
	}

	userTrends := s.extractUserTrends(qas, from, to, datetimeFormat, nextFunc)

	userTrendsHourly := s.extractUserTrendsHourly(qas)

	satisfactionSurvey := s.extractSatisfactionSurvey(qas)

	askHistoryDaily := s.extractAskHistoryDaily(qas, from, to, datetimeFormat, nextFunc)

	llmTokenHistoryDaily := s.extractLLMTokenHistoryDaily(qas, from, to, datetimeFormat, nextFunc)

	keywordTrends := s.extractKeywordTrends(qas, from, to, datetimeFormat, nextFunc)
	sort.Slice(keywordTrends.Summary.Ranking, func(i, j int) bool {
		return keywordTrends.Summary.Ranking[i].Count > keywordTrends.Summary.Ranking[j].Count
	})
	sort.Slice(keywordTrends.TimeBased, func(i, j int) bool {
		return keywordTrends.TimeBased[i].DateTime < keywordTrends.TimeBased[j].DateTime
	})
	for i := 0; i < len(keywordTrends.TimeBased); i++ {
		sort.Slice(keywordTrends.TimeBased[i].Ranking, func(j, k int) bool {
			return keywordTrends.TimeBased[i].Ranking[j].Count > keywordTrends.TimeBased[i].Ranking[k].Count
		})
	}

	return &model.Statistics{
		UserTrends:         *userTrends,
		UserTrendsHourly:   *userTrendsHourly,
		SatisfactionSurvey: *satisfactionSurvey,
		AskHistory:         *askHistoryDaily,
		LLMTokenHistory:    *llmTokenHistoryDaily,
		KeywordTrends:      *keywordTrends,
	}, nil
}
