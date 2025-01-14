package model

type FileReference struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
}
