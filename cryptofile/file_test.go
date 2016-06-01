package foo

import (
	"bytes"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

type PlainTextBlock struct {
	BlockSz int
}

func (pt *PlainTextBlock) BlockSize() int          { return pt.BlockSz }
func (pt *PlainTextBlock) Encrypt(dst, src []byte) {}
func (pt *PlainTextBlock) Decrypt(dst, src []byte) {}

func TestReadWrite(t *testing.T) {

	random := rand.New(rand.NewSource(4532))

	for _, blocksz := range []int{1, 2, 8, 32} {
		for sz := 0; sz < 1000; sz++ {
			var buf bytes.Buffer

			data := make([]byte, sz, sz)
			_, err := io.ReadFull(random, data)
			if err != nil {
				t.Fatal(err)
			}

			block := &PlainTextBlock{BlockSz: blocksz}
			w := NewWriter(block, &buf)

			ncopied := 0
			for ncopied != len(data) {
				amnt := rand.Int() % (blocksz * 3)
				if ncopied+amnt > len(data) {
					amnt = len(data) - ncopied
				}
				n, err := w.Write(data[ncopied : ncopied+amnt])
				if err != nil {
					t.Fatal(err)
				}
				ncopied += n
			}

			err = w.Close()
			if err != nil {
				t.Fatal(err)
			}

			if buf.Len()%blocksz != 0 {
				t.Fatal("len is not a multiple of block size")
			}

			rdr := NewReader(bytes.NewReader(buf.Bytes()), block, int64(buf.Len()))
			result := make([]byte, len(data), len(data))
			nread := 0
			for nread != len(data) {
				amnt := rand.Int() % (blocksz * 3)
				if nread+amnt > len(data) {
					amnt = len(data) - nread
				}
				n, err := io.ReadFull(rdr, result[nread:nread+amnt])
				if err != nil {
					t.Fatal(err)
				}
				nread += n
			}

			n, err := io.ReadFull(rdr, make([]byte, 10000, 10000))
			if n != 0 || err != io.EOF {
				t.Fatal("expected EOF")
			}

			if !reflect.DeepEqual(data, result) {
				t.Fatalf("data differs: %v != %v", result, data)
			}
		}
	}

}
