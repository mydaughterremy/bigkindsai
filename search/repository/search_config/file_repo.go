package search_config

import (
	"context"
	"os"

	"bigkinds.or.kr/proto"
)

type FileSearchConfigRepository struct {
	FilePath string
}

func (r *FileSearchConfigRepository) GetConfig(ctx context.Context) (*proto.SearchConfig, error) {
	// open search_config.json
	b, err := os.ReadFile(r.FilePath)
	if err != nil {
		return nil, err
	}

	config, err := UnmarshalSearchConfig(b)
	if err != nil {
		return nil, err
	}

	// validate config
	err = config.Validate()
	if err != nil {
		return nil, err
	}

	return config, nil
}
