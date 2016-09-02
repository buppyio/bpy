package sig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/buppyio/bpy"
	"strings"
)

var (
	ErrInvalidSignedHash = errors.New("invalid signed hash")
	ErrSigFailed         = errors.New("signature check failed")
)

func ParseSignedHash(k *bpy.Key, signedAndSig string) ([32]byte, error) {
	idx := strings.LastIndex(signedAndSig, ":")
	if idx == -1 {
		return [32]byte{}, ErrInvalidSignedHash
	}
	signedString := signedAndSig[:idx]
	signedMac, err := hex.DecodeString(signedAndSig[idx+1:])
	if err != nil {
		return [32]byte{}, ErrInvalidSignedHash
	}
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(signedString))
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(signedMac, expectedMAC) {
		return [32]byte{}, ErrSigFailed
	}
	var hash [32]byte
	_, err = hex.Decode(hash[:], []byte(signedString))
	return hash, err
}

func SignHash(k *bpy.Key, hash [32]byte) string {
	hashHex := hex.EncodeToString(hash[:])
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(hashHex))
	hashMac := mac.Sum(nil)
	signedHash := fmt.Sprintf("%s:%s", hashHex, hex.EncodeToString(hashMac))
	return signedHash
}
