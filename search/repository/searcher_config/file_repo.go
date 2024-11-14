package searcher_config

import (
	"context"
	"os"

	"bigkinds.or.kr/proto/searcher"
)

type FileSearcherConfigRepository struct {
	FilePath string
}

func (r *FileSearcherConfigRepository) GetConfig(ctx context.Context) (*searcher.SearcherConfig, error) {
	// open search_config.json
	b, err := os.ReadFile(r.FilePath)
	if err != nil {
		return nil, err
	}

	config, err := UnmarshalSearcherConfig(b)
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
