package main

import (
	"bigkinds.or.kr/backend/model"
)

type Query struct {
	QA           *model.QA
	ShouldCreate bool
}

func mergeQA(src *model.QA, dst *model.QA) *model.QA {
	if src == nil {
		return dst
	}

	if dst == nil {
		return src
	}

	if src.SessionID != "" {
		dst.SessionID = src.SessionID
	}
	if src.JobGroup != "" {
		dst.JobGroup = src.JobGroup
	}
	if src.ChatID != "" {
		dst.ChatID = src.ChatID
	}
	if src.Question != "" {
		dst.Question = src.Question
	}
	if src.Answer != "" {
		dst.Answer = src.Answer
	}
	if src.TokenCount != 0 {
		dst.TokenCount = src.TokenCount
	}
	if len(src.References) > 0 {
		dst.References = src.References
	}
	if len(src.Keywords) > 0 {
		dst.Keywords = src.Keywords
	}
	if len(src.RelatedQueries) > 0 {
		dst.RelatedQueries = src.RelatedQueries
	}
	dst.UpdatedAt = src.UpdatedAt

	return dst
}

func mergeQueries(queries []*Query) map[string]*Query {
	updates := make(map[string]*Query)
	for _, query := range queries {
		qa := query.QA
		id := query.QA.ID

		if _, ok := updates[id]; !ok {
			updates[id] = query
			continue
		}

		merged := mergeQA(qa, updates[id].QA)
		updates[id].QA = merged
		updates[id].ShouldCreate = updates[id].ShouldCreate || query.ShouldCreate
	}

	return updates
}
