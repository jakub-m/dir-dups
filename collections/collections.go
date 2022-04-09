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

func TransformSlice[S, T any](ss []S, fn func(S) T) []T {
	tt := []T{}
	for _, s := range ss {
		tt = append(tt, fn(s))
	}
	return tt
}

func Uniq[T comparable](ss []T) []T {
	m := make(map[T]bool)
	for _, s := range ss {
		m[s] = true
	}
	u := []T{}
	for k := range m {
		u = append(u, k)
	}
	return u
}

func FilterSlice[T any](ss []T, fn func(T) bool) []T {
	out := []T{}
	for _, s := range ss {
		if fn(s) {
			out = append(out, s)
		}
	}
	return out
}

func ReverseSlice[T any](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func Deduplicate[T comparable](values []T) []T {
	set := make(map[T]bool)
	for _, val := range values {
		set[val] = true
	}
	out := []T{}
	for val := range set {
		out = append(out, val)
	}
	return out
}
