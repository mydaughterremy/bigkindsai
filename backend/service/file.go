package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
	"github.com/google/uuid"
)

type FileService struct {
	FileRepository *repository.FileRepository
	QARepository   *repository.QARepository
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponseData struct {
	Embedding []float64 `json:"embedding"`
}

type EmbeddingResponseUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type EmbeddingResponse struct {
	Data  []EmbeddingResponseData `json:"data"`
	Usage EmbeddingResponseUsage  `json:"usage"`
}

func NewFileService() (*FileService, error) {
	return &FileService{}, nil
}

func (f *FileService) CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("verctors must have same length. leng(a)=%d, leng(b)=%d", len(a), len(b))
	}

	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("vector must not be empty")
	}

	var (
		dotProduct float64
		normA      float64
		normB      float64
	)

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("vertor with zero mgnitude detected")
	}

	similarity := dotProduct / (normA * normB)

	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity, nil
}

func (f *FileService) GetEmbedding(content string) ([]float64, int, error) {
	apiKey := "up_cyM2Ajc0N3iYvaDIAIS4XtOaElBfC"
	url := "https://api.upstage.ai/v1/solar/embeddings"

	reqBody := EmbeddingRequest{
		Model: "embedding-query",
		Input: content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("embedding response unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, 0, err
	}

	return embeddingResp.Data[0].Embedding, embeddingResp.Usage.TotalTokens, nil
}

func (f *FileService) GetFileContent(fileId string) ([]byte, error) {
	fp := filepath.Join("./upload", fileId)
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (f *FileService) GetFiles(ctx context.Context, uploadId string) ([]*model.File, error) {
	files, err := f.FileRepository.GetFiles(ctx, uploadId)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (f *FileService) GetUploadId(ctx context.Context, chatId string) string {
	uploadId := f.FileRepository.GetUploadId(ctx, chatId)
	return uploadId
}

func (f *FileService) WriteUploadFiles(ctx context.Context, multipleFileResponse *model.MultipleFileResponse) error {
	for _, uploadFile := range multipleFileResponse.UploadFiles {
		file := &model.File{
			ID:       uploadFile.ID,
			Filename: uploadFile.Filename,
			UploadID: multipleFileResponse.UploadId,
		}
		err := f.FileRepository.CreateFile(ctx, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileService) WriteUploadQA(ctx context.Context, multipleFileResponse *model.MultipleFileResponse) (*model.QA, error) {
	qa, err := f.QARepository.CreateQA(ctx, &model.QA{
		ID:        uuid.New().String(),
		ChatID:    multipleFileResponse.ChatId,
		SessionID: "",
		JobGroup:  "",
		UploadID:  multipleFileResponse.UploadId,
		Pages:     multipleFileResponse.TotalPages,
		Filenames: multipleFileResponse.Filenames,
	})

	if err != nil {
		return nil, err
	}

	return qa, nil
}
