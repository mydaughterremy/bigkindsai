package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/service"
	"github.com/google/uuid"
)

type FileHandler struct {
	service   *service.FileService
	UploadDir string
	MaxSize   int64 // maximum file size in bytes
	MaxNum    int64
}

type MultipleFileResponse struct {
	fileIds []string
}

func (f *FileHandler) MultipleFileUpload(w http.ResponseWriter, r *http.Request) {
	maxCap := f.MaxNum * f.MaxSize
	r.Body = http.MaxBytesReader(w, r.Body, maxCap)

	err := r.ParseMultipartForm(maxCap)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, fmt.Errorf("there is no file in request"))
		return
	}

	uploadedFiles := []string{}

	for _, fileHeader := range files {
		filename := uuid.New().String()
		filepath := filepath.Join(f.UploadDir, filename)
		src, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer src.Close()

		dst, err := os.Create(filepath)
		if err != nil {
			continue
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			continue
		}
		uploadedFiles = append(uploadedFiles, filename)

	}

	if len(uploadedFiles) == 0 {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, fmt.Errorf("failed to upload any files"))
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, &MultipleFileResponse{
		fileIds: uploadedFiles,
	})

}
