package utils

type Pair[T interface{}] struct {
	First, Second *T
}
