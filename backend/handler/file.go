package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FileHandler struct {
	FileService       *service.FileService
	ChatService       *service.ChatService
	CompletionService *service.CompletionService

	UploadDir string
	MaxSize   int64 // maximum file size in bytes
	MaxNum    int64
}

type FileResponse struct {
	FileId string `json:"file_id"`
}

type DocParserResponse struct {
	Content DocParserResponseContent `json:"content"`
	Usages  DocParserResponseUsage   `json:"usage"`
}

type DocParserResponseContent struct {
	Text string `json:"text"`
}

type DocParserResponseUsage struct {
	Pages int `json:"pages"`
}

type DocParserResult struct {
	UploadId string `json:"upload_id"`
	FileId   string `json:"file_id"`
	Content  string `json:"content"`
	Usage    int    `json:"usage"`
}

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id"`
	Stamp   string `json:"stamp"`
}

type ConversionStatus struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	ErrorCode int    `json:"errorCode"`
}
type FileContent struct {
	FileId   string
	Filename string
	Content  string
}

type FileChunk struct {
	FileId    string
	Score     float64
	Chunk     string
	Filename  string
	Embedding []float64
}

func (f *FileHandler) GetUploadDir() string {
	return f.UploadDir
}

func (f *FileHandler) FileUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(f.MaxSize)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	apiKey := "up_cyM2Ajc0N3iYvaDIAIS4XtOaElBfC"
	url := "https://api.upstage.ai/v1/document-ai/document-parse"

	var reqBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&reqBody)

	err = multipartWriter.WriteField("ocr", "auto")
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	err = multipartWriter.WriteField("output_formats", `["text"]`)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	err = multipartWriter.WriteField("model", "document-parse")
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	formFile, err := multipartWriter.CreateFormFile("document", handler.Filename)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_, err = io.Copy(formFile, file)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	multipartWriter.Close()

	req, err := http.NewRequest("POST", url, &reqBody)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	var docResp DocParserResponse
	err = json.Unmarshal(respBytes, &docResp)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	uploadId := uuid.New().String()
	fileId := uuid.New().String()
	docRes := &DocParserResult{
		UploadId: uploadId,
		FileId:   fileId,
		Content:  docResp.Content.Text,
		Usage:    docResp.Usages.Pages,
	}
	filepath := filepath.Join(f.UploadDir, fileId)
	dst, err := os.Create(filepath)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}
	defer dst.Close()

	if _, err := dst.Write([]byte(docResp.Content.Text)); err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, docRes)

}

func (f *FileHandler) FileConvert(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Step 1: Upload file
	uploadResp, err := f.uploadFile(file)
	if err != nil {
		http.Error(w, "Upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 2: Check conversion status
	for {
		status, err := f.checkConversion(uploadResp.ID)
		if err != nil {
			http.Error(w, "Status check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if status.Status == "S" {
			break
		}
		time.Sleep(time.Second)
	}
	// Step 3: Download converted file
	convertedFile, err := f.downloadFile(uploadResp.ID, uploadResp.Stamp)
	if err != nil {
		http.Error(w, "Download failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 4: Send response
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=converted.pdf")
	w.Write(convertedFile)
}

func (f *FileHandler) uploadFile(file multipart.File) (*UploadResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "file")
	if err != nil {
		return nil, err
	}
	io.Copy(part, file)

	writer.WriteField("convertType", "PDF")
	writer.Close()

	req, err := http.NewRequest("POST", "http://converter.infraware.net/file/upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, err
	}

	return &uploadResp, nil
}

func (f *FileHandler) checkConversion(id string) (*ConversionStatus, error) {
	resp, err := http.Get(fmt.Sprintf("http://converter.infraware.net/file/convert/%s", id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status ConversionStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (f *FileHandler) downloadFile(id, stamp string) ([]byte, error) {
	url := fmt.Sprintf("http://converter.infraware.net/file/download/%s.pdf?stamp=%s", id, stamp)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (f *FileHandler) MultipleFileUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var fileNames []string
	chatId := chi.URLParam(r, "chat_id")
	if chatId == "" {
		// chatId = uuid.New().String()
		chat, err := f.ChatService.CreateChat(ctx, "")
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		chatId = chat.ID.String()
	}

	apiKey := "up_cyM2Ajc0N3iYvaDIAIS4XtOaElBfC"
	url := "https://api.upstage.ai/v1/document-ai/document-parse"
	maxCap := f.MaxNum * f.MaxSize
	r.Body = http.MaxBytesReader(w, r.Body, maxCap)

	fmt.Println("maxCap", string(int(maxCap)))

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

	uploadedFiles := []model.UploadFile{}
	var totalPages int
	for _, fileHeader := range files {
		fmt.Println("Filename: " + fileHeader.Filename)
		fileId := uuid.New().String()
		filepath := filepath.Join(f.UploadDir, fileId)
		src, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer src.Close()

		var reqBody bytes.Buffer
		multipartWriter := multipart.NewWriter(&reqBody)

		err = multipartWriter.WriteField("ocr", "auto")
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		err = multipartWriter.WriteField("output_formats", `["text"]`)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		err = multipartWriter.WriteField("model", "document-parse")
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		formFile, err := multipartWriter.CreateFormFile("document", fileHeader.Filename)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		_, err = io.Copy(formFile, src)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		multipartWriter.Close()

		req, err := http.NewRequest("POST", url, &reqBody)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		var docResp DocParserResponse
		err = json.Unmarshal(respBytes, &docResp)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}

		totalPages += docResp.Usages.Pages

		dst, err := os.Create(filepath)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		defer dst.Close()
		if _, err := dst.Write([]byte(docResp.Content.Text)); err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		uploadedFiles = append(uploadedFiles, model.UploadFile{
			ID:       fileId,
			Filename: fileHeader.Filename,
		})

		fileNames = append(fileNames, fileHeader.Filename)

	}

	if len(uploadedFiles) == 0 {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, fmt.Errorf("failed to upload any files"))
		return
	}

	uploadId := uuid.New().String()

	multipleFileResponse := &model.MultipleFileResponse{
		ChatId:      chatId,
		UploadId:    uploadId,
		TotalPages:  totalPages,
		UploadFiles: uploadedFiles,
		Filenames:   fileNames,
	}

	err = f.FileService.WriteUploadFiles(ctx, multipleFileResponse)
	if err != nil {
		// file delete
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	qa, err := f.FileService.WriteUploadQA(ctx, multipleFileResponse)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, qa)

}
func (f *FileHandler) CreateChatCompletionFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateChatCompletionFileRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	var fc []*FileChunk

	chatId := chi.URLParam(r, "chat_id")
	uploadId := f.FileService.GetUploadId(ctx, chatId)
	if uploadId == "" {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, fmt.Errorf("there is no uploadfile in this chatid"))
		return
	}

	files, err := f.FileService.GetFiles(ctx, uploadId)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, fmt.Errorf("error get files from uploadid"))
		return
	}

	var fx []*FileContent

	for _, file := range files {
		fileContent, err := f.FileService.GetFileContent(file.ID)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, fmt.Errorf("error read file content"))
			return
		}
		fx = append(fx, &FileContent{
			FileId:   file.ID,
			Filename: file.Filename,
			Content:  string(fileContent),
		})
	}

	chunkSize := 500

	var embeddingTokens int

	for _, file := range fx {
		s := file.Content
		for i := 0; i < len(s); i += chunkSize {
			end := i + chunkSize
			if end > len(s) {
				end = len(s)
			}
			c := s[i:end]

			e, t, err := f.FileService.GetEmbedding(c)
			if err != nil {
				_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
				return
			}

			embeddingTokens += t

			fc = append(fc, &FileChunk{
				FileId:    file.FileId,
				Chunk:     c,
				Filename:  file.Filename,
				Embedding: e,
			})
		}

	}

	messageE, t, err := f.FileService.GetEmbedding(req.Message)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	embeddingTokens += t

	for _, c := range fc {
		s, err := f.FileService.CosineSimilarity(messageE, c.Embedding)
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		}
		c.Score = s
	}

	sort.Slice(fc, func(i, j int) bool {
		return fc[i].Score > fc[j].Score
	})

	slog.Info(fmt.Sprintf("best score: %f", fc[0].Score))

	slog.Info("===== file handling is finished")

	// slog.Info("===== do after file handling")

	topk := 5

	if len(fc) < topk+1 {
		topk = len(fc)
	}

	var messages []*model.Message
	var fileReferences []model.FileReference

	for _, file := range fc[:topk] {
		messages = append(messages, &model.Message{
			Role:    "assistant",
			Content: file.Chunk,
		})

		fileReferences = append(fileReferences, model.FileReference{
			FileName: file.Filename,
			Content:  strings.ToValidUTF8(file.Chunk, ""),
		})
	}

	messages = append(messages, &model.Message{
		Role:    "user",
		Content: req.Message,
	})

	var chatChannel chan *service.CreateChatCompletionResult
	chatChannel, err = f.CompletionService.CreateChatCompletionFile(ctx, &service.CreateChatCompletionParameter{
		ChatID:   chatId,
		Session:  req.Session,
		JobGroup: req.JobGroup,
		Messages: messages,
	}, fileReferences)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Add("Content-Type", "text/event-stream;charset=utf-8")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-chatChannel:
				if !ok {
					return
				}
				if result.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, result.Error)
					return
				} else if result.Done {
					_ = response.WriteStreamResponse(w, []byte("[DONE]"))
					return
				}

				body, err := json.Marshal(result.Completion)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, err)
					return
				}
				_ = response.WriteStreamResponse(w, body)
			}
		}

	}()

	wg.Wait()

	// _ = response.WriteJsonResponse(w, r, http.StatusOK, fc)
}

func (f *FileHandler) GetUploadId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatId := chi.URLParam(r, "chat_id")
	uploadId := f.FileService.GetUploadId(ctx, chatId)

	_ = response.WriteJsonResponse(w, r, http.StatusOK, uploadId)
}
