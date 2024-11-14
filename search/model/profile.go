package model

type SearchProfile struct {
	SearcherTime float64
	RerankerTime float64
}

type MSearchProfile struct {
	TotalElapsedTime float64
	SearchProfile    []*SearchProfile
	AggregatorTime   float64
}

type SearchServiceProfile struct {
	TotalElapsedTime    float64
	AuthTime            float64
	GetSearchConfigTime float64
	SearchTime          float64
	SearchProfile       *SearchProfile
	MSearchProfile      *MSearchProfile
}
