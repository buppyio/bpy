package bpack

type IndexEnt struct {
	Key    string
	Offset uint64
}

type Index []IndexEnt

func (idx Index) Len() int           { return len(idx) }
func (idx Index) Swap(i, j int)      { t := idx[i]; idx[i] = idx[j]; idx[j] = t }
func (idx Index) Less(i, j int) bool { return keycmp(idx[i].Key, idx[j].Key) }

func keycmp(l, r string) bool {
	if len(l) != len(r) {
		if len(l) < len(r) {
			return true
		} else {
			return false
		}
	}

	for k := range l {
		if l[k] != r[k] {
			if l[k] < r[k] {
				return true
			} else {
				return false
			}
		}
	}
	return false
}
