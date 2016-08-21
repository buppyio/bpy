package cryptofile

import (
	"fmt"
)

type ctrState struct {
	Iv  []byte
	Vec []byte
}

func newCtrState(iv []byte) *ctrState {
	ctr := &ctrState{
		Iv:  make([]byte, len(iv), len(iv)),
		Vec: make([]byte, len(iv), len(iv)),
	}
	copy(ctr.Iv, iv)
	copy(ctr.Vec, iv)
	return ctr
}

func (ctr *ctrState) Reset() {
	copy(ctr.Vec, ctr.Iv)
}

func (ctr *ctrState) Add(val uint64) {
	idx := len(ctr.Vec) - 1
	carry := uint64(0)
	for val != 0 || carry != 0 {
		b := val & 0xff
		val = val >> 8

		if idx < 0 {
			break
		}

		newb := uint64(ctr.Vec[idx]) + b + carry
		ctr.Vec[idx] = byte(newb)

		if newb&(1<<8) != 0 {
			carry = 1
		} else {
			carry = 0
		}

		idx--
	}
}

func Xor(dest, src []byte) {
	if len(dest) != len(src) {
		panic(fmt.Sprintf("Xor with different length buffers %v != %v", len(dest), len(src)))
	}
	for idx := range src {
		dest[idx] = dest[idx] ^ src[idx]
	}
}
