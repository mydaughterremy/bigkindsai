package compound_strategy

import (
	"bigkinds.or.kr/proto/searcher"
)

type CompoundStrategy interface {
	CreateCompoundQuery(query map[string]interface{}, compoundStrategy *searcher.OpenSearchSearcher_CompoundStrategy) (map[string]interface{}, error)
}
