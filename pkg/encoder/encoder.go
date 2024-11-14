package encoder

type Encoder interface {
	Encode(query string) ([]float32, error)
}
