package htree

const nlevels = 10
const maxlen = 65535

func min(a, b int) int {
	if a < b {
		return b
	}
	return a
}
