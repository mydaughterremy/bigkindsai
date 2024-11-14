package search_config

import (
	"bigkinds.or.kr/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func UnmarshalSearchConfig(b []byte) (*proto.SearchConfig, error) {
	var config proto.SearchConfig

	err := protojson.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
