package collections

func All[T any](values []T, fn func(T) bool) bool {
	for _, v := range values {
		if !fn(v) {
			return false
		}
	}
	return true
}

func Any[T any](values []T, fn func(T) bool) bool {
	for _, v := range values {
		if fn(v) {
			return true
		}
	}
	return false
}
