package sig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/buppyio/bpy"
	"strconv"
	"strings"
)

var (
	ErrInvalidSignedHash = errors.New("invalid signed hash")
	ErrSigFailed         = errors.New("signature check failed")
	ErrCorruptVersion    = errors.New("version in signature is corrupt")
)

func ParseSignedHash(k *bpy.Key, signedAndSig string) (int64, [32]byte, error) {
	firstSep := strings.Index(signedAndSig, ":")
	if firstSep == -1 {
		return -1, [32]byte{}, ErrInvalidSignedHash
	}
	lastSep := strings.LastIndex(signedAndSig, ":")
	if lastSep == -1 {
		return -1, [32]byte{}, ErrInvalidSignedHash
	}
	if firstSep == lastSep {
		return -1, [32]byte{}, ErrInvalidSignedHash
	}
	versionStr := signedAndSig[:firstSep]
	hashStr := signedAndSig[firstSep+1 : lastSep]
	signedStr := signedAndSig[:lastSep]
	sigMac, err := hex.DecodeString(signedAndSig[lastSep+1:])
	if err != nil {
		return -1, [32]byte{}, ErrInvalidSignedHash
	}
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(signedStr))
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(sigMac, expectedMAC) {
		return -1, [32]byte{}, ErrSigFailed
	}
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return -1, [32]byte{}, ErrCorruptVersion
	}
	var hash [32]byte
	_, err = hex.Decode(hash[:], []byte(hashStr))
	return version, hash, err
}

func ParseVersion(signedHash string) (int64, error) {
	firstSep := strings.Index(signedHash, ":")
	if firstSep == -1 {
		return -1, ErrInvalidSignedHash
	}
	versionStr := signedHash[:firstSep]
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return -1, ErrCorruptVersion
	}
	return version, nil
}

func SignHash(k *bpy.Key, version int64, hash [32]byte) string {
	toSign := fmt.Sprintf("%d:%s", version, hex.EncodeToString(hash[:]))
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(toSign))
	hashMac := mac.Sum(nil)
	signedHash := fmt.Sprintf("%s:%s", toSign, hex.EncodeToString(hashMac))
	return signedHash
}
