package searcher_config

import (
	"bigkinds.or.kr/proto/searcher"
	"google.golang.org/protobuf/encoding/protojson"
)

func UnmarshalSearcherConfig(b []byte) (*searcher.SearcherConfig, error) {
	var config searcher.SearcherConfig

	err := protojson.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
