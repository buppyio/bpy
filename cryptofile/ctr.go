package cryptofile

type ctr struct {
	Iv  []byte
	Vec []byte
}

func newCtr(iv []byte) *ctr {
	c := &ctr{
		Iv:  make([]byte, len(iv), len(iv)),
		Vec: make([]byte, len(iv), len(iv)),
	}
	copy(c.Iv, iv)
	copy(c.Vec, iv)
	return c
}

func (c *ctr) Reset() {
	copy(c.Vec, c.Iv)
}

func (c *ctr) Add(val uint64) {
	idx := len(c.Vec) - 1
	carry := uint64(0)
	for val != 0 || carry != 0 {
		b := val & 0xff
		val = val >> 8

		if idx < 0 {
			break
		}

		newb := uint64(c.Vec[idx]) + b + carry
		c.Vec[idx] = byte(newb)

		if newb&(1<<8) != 0 {
			carry = 1
		} else {
			carry = 0
		}

		idx--
	}
}
