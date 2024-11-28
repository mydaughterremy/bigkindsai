package reranker

import (
	"bigkinds.or.kr/proto"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
)

type SolarEmbedding struct {
	Client  *http.Client
	ApiKey  string
	BaseURL string
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (s *SolarEmbedding) GetEmbeddings(text string) ([]float32, error) {
	model := os.Getenv("UPSTAGE_EMBEDDING_MODEL")
	// API 요청 구조체
	requestBody := map[string]interface{}{
		"input": text,
		"model": model,
	}
	response, err := s.RequestEmbedding(requestBody)
	if response.Data == nil {
		return nil, err
	}
	return response.Data[0].Embedding, err
}

func (s *SolarEmbedding) GetEmbeddingsList(text []string) ([][]float32, error) {
	model := os.Getenv("SOLAR_EMBEDDING_MODEL")
	// API 요청 구조체
	requestBody := map[string]interface{}{
		"input": text,
		"model": model,
	}

	response, err := s.RequestEmbedding(requestBody)
	datas := make([][]float32, 0, len(response.Data))
	for _, data := range response.Data {
		embedding := data.Embedding
		datas = append(datas, embedding)
	}
	return datas, err
}
func (s *SolarEmbedding) RequestEmbedding(requestBody map[string]interface{}) (EmbeddingResponse, error) {
	// JSON 인코딩
	jsonData, err := json.Marshal(requestBody)
	var response EmbeddingResponse

	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %v", err)
	}

	// API 요청 생성 - URL 경로 수정
	req, err := http.NewRequest("POST", s.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return response, fmt.Errorf("failed to create request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)

	// 요청 전송
	resp, err := s.Client.Do(req)
	if err != nil {
		return response, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 바디 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %v", err)
	}

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// JSON 응답 파싱
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 결과 확인
	if len(response.Data) == 0 {
		return response, fmt.Errorf("no embeddings returned")
	}

	return response, nil
}

// 코사인 유사도 계산 함수
func cosineSimilarity(a []float32, b []float32) float32 {
	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Reranker 구현
type SolarReranker struct {
	Embedder *SolarEmbedding
}

func (r *SolarReranker) Rerank(query string, documents []*proto.Item, k int32) ([]*proto.Item, error) {
	// 쿼리 임베딩 생성
	queryEmbedding, err := r.Embedder.GetEmbeddings(query)
	if err != nil {
		return nil, err
	}

	type itemScore struct {
		item  *proto.Item
		score float32
	}

	if len(documents) <= 0 {
		return nil, err
	}

	scores := make([]itemScore, len(documents))
	contents := make([]string, 0, len(documents))

	for _, doc := range documents {
		// attributes에서 임베딩할 텍스트 추출 (예: content 필드)
		content := doc.Attributes.Fields["content"].GetStringValue()
		contents = append(contents, content)
	}

	// 문서 텍스트 임베딩 생성
	docEmbeddings, err := r.Embedder.GetEmbeddingsList(contents)
	if err != nil {
		return nil, err
	}

	// 유사도 계산
	for i, docEmbedding := range docEmbeddings {
		similarity := cosineSimilarity(queryEmbedding, docEmbedding)
		scores[i] = itemScore{
			item:  documents[i],
			score: similarity,
		}
	}

	// 유사도 기준으로 정렬
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 정렬된 결과 생성
	result := make([]*proto.Item, len(scores))
	for i, score := range scores {
		// 기존 Item을 유지하면서 새로운 score만 업데이트
		result[i] = &proto.Item{
			Id:         score.item.Id,
			Attributes: score.item.Attributes,
			Score:      score.score, // 새로 계산된 유사도 점수로 업데이트
		}
	}

	return result, nil
}
