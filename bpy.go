package bpy

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

type CStore interface {
	Flush() error
	Put([]byte) ([32]byte, error)
	Get([32]byte) ([]byte, error)
	Close() error
}

type Key struct {
	CipherKey [32]byte
	HmacKey   [32]byte
	Id        [32]byte
}

func NewKey() (Key, error) {
	var k Key

	_, err := io.ReadFull(rand.Reader, k.CipherKey[:])
	if err != nil {
		return k, fmt.Errorf("error generating cipher key: %s", err.Error())
	}

	_, err = io.ReadFull(rand.Reader, k.HmacKey[:])
	if err != nil {
		return k, fmt.Errorf("error generating hmac key: %s", err.Error())
	}

	_, err = io.ReadFull(rand.Reader, k.Id[:])
	if err != nil {
		return k, fmt.Errorf("error generating id: %s", err.Error())
	}

	return k, nil
}

func WriteKey(w io.Writer, k *Key) error {
	j, err := json.Marshal(k)
	if err != nil {
		return fmt.Errorf("error marshaling key: %s", err.Error())
	}
	_, err = w.Write(j)
	if err != nil {
		return fmt.Errorf("error writing key: %s", err.Error())
	}
	return nil
}

func ReadKey(r io.Reader) (Key, error) {
	var k Key

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return k, fmt.Errorf("error reading key: %s", err.Error())
	}
	err = json.Unmarshal(data, &k)
	if err != nil {
		return k, fmt.Errorf("error unmarshalling key: %s", err.Error())
	}
	return k, nil
}

func ParseHash(hashstr string) ([32]byte, error) {
	var hash [32]byte
	if len(hashstr) != 64 {
		return hash, errors.New("bad length")
	}
	_, err := hex.Decode(hash[:], []byte(hashstr))
	if err != nil {
		return hash, err
	}
	return hash, nil
}

func RandomFileName() (string, error) {
	namebuf := [32]byte{}
	_, err := io.ReadFull(rand.Reader, namebuf[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(namebuf[:]), nil
}

type BufferedWriteCloser struct {
	W io.WriteCloser
	B *bufio.Writer
}

func (bwc *BufferedWriteCloser) Write(buf []byte) (int, error) {
	return bwc.B.Write(buf)
}

func (bwc *BufferedWriteCloser) Close() error {
	err := bwc.B.Flush()
	if err != nil {
		return err
	}
	return bwc.W.Close()
}

func NewRootVersion() (string, error) {
	r := [sha256.Size]byte{}
	_, err := io.ReadFull(rand.Reader, r[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(r[:]), nil
}

func NewGCGeneration() (string, error) {
	r := [sha256.Size]byte{}
	_, err := io.ReadFull(rand.Reader, r[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(r[:]), nil
}

func NextRootVersion(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func NextGCGeneration(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
