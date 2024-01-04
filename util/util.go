package util

import (
	"fmt"
)

func Filter[T any](vs []T, f func(T) bool) []T {
	filtered := make([]T, 0)
	for _, v := range vs {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func StringsJoin(elems []fmt.Stringer, sep string) string {
	var res string
	for i := 0; i < len(elems); i++ {
		res += elems[i].String()
		if len(elems) != i+1 {
			res += ", "
		}
	}
	return res
}

func Exists[a comparable](elems []a, elem a) bool {
	for _, e := range elems {
		if e == elem {
			return true
		}
	}
	return false
}
