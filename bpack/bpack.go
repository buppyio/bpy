package bpack

type IndexEnt struct {
	Key    string
	Size   uint32
	Offset uint64
}

type Index []IndexEnt

func (idx Index) Len() int           { return len(idx) }
func (idx Index) Swap(i, j int)      { idx[i], idx[j] = idx[j], idx[i] }
func (idx Index) Less(i, j int) bool { return KeyCmp(idx[i].Key, idx[j].Key) < 0 }

func KeyCmp(l, r string) int {
	if l == r {
		return 0
	}
	if l < r {
		return -1
	}
	return +1
}

func (idx Index) Search(key string) (int, bool) {
	lo := 0
	hi := len(idx) - 1
	for lo <= hi {
		mid := (hi + lo) / 2
		switch KeyCmp(key, idx[mid].Key) {
		case -1:
			hi = mid - 1
		case 1:
			lo = mid + 1
		case 0:
			return mid, true
		}
	}
	return -1, false
}
