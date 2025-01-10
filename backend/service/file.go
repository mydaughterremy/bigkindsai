package service

type FileService struct {
}

func NewFileService() (*FileService, error) {
	return &FileService{}, nil
}
