package main

func Filter[T any](vs []T, f func(T) bool) []T {
	filtered := make([]T, 0)
	for _, v := range vs {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func HasAlias(a Arg) bool {
	return a.alias != ""
}

func SplitTokensBy(tokens []token, kind tokKind) [][]token {
	newtokens := make([][]token, 0)
	i := 0
	for _, t := range tokens {
		if t.kind == kind {
			i++
			continue
		}
		if len(newtokens) == i {
			newtokens = append(newtokens, []token{t})
		} else {
			newtokens[i] = append(newtokens[i], t)
		}
	}
	return newtokens
}

// TODO: gotta be a better way...
func StringsJoin(elems []Arg, sep string) string {
	var res string
	for i := 0; i < len(elems); i++ {
		res += elems[i].String()
		if len(elems) != i+1 {
			res += ", "
		}
	}
	return res
}
