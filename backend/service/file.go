package service

import (
	"context"
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

func NewFileService() (*FileService, error) {
	return &FileService{}, nil
}

func (f *FileService) GetEmbedding(content string) ([]float64, int, error) {
	// apiKey := "up_cyM2Ajc0N3iYvaDIAIS4XtOaElBfC"
	// url := "https://api.upstage.ai/v1/solar/embeddings"

	// var reqBody bytes.Buffer

	return nil, 0, nil
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
