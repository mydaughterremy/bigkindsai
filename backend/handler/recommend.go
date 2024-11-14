package handler

import (
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
)

type GetRecommendedQuestionsResponse struct {
	Questions []string `json:"questions"`
}

func GetRecommendedQuestions(w http.ResponseWriter, r *http.Request) {
	// return response with fixed strings as json format
	sampleResponse := GetRecommendedQuestionsResponse{
		Questions: []string{
			"빅카인즈AI가 뭐야?",
			"최근 생성형 인공지능의 발전에 대해 알려줘",
			"최근 생성형 인공지능의 발전에 대한 정보를 서론, 본론, 결론의 형태로 언론사 기자가 보도하는 스타일로 작성해줘",
		},
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, sampleResponse)
}
