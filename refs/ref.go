package refs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buppyio/bpy"
	"strings"
)

var (
	ErrInvalidRefString = errors.New("invalid ref string")
	ErrSigFailed        = errors.New("def signature check failed")
)

type Ref struct {
	Root [32]byte
}

func ParseRef(k *bpy.Key, signedRef string) (Ref, error) {
	idx := strings.LastIndex(signedRef, ":")
	if idx == -1 {
		return Ref{}, ErrInvalidRefString
	}
	refString := signedRef[:idx]
	refMac, err := hex.DecodeString(signedRef[idx+1:])
	if err != nil {
		return Ref{}, ErrInvalidRefString
	}
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(refString))
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(refMac, expectedMAC) {
		return Ref{}, ErrSigFailed
	}
	ref := Ref{}
	err = json.Unmarshal([]byte(refString), &ref)
	return ref, err
}

func SerializeAndSign(k *bpy.Key, ref Ref) (string, error) {
	refBytes, err := json.Marshal(ref)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(refBytes))
	refMac := mac.Sum(nil)
	signedRef := fmt.Sprintf("%s:%s", string(refBytes), hex.EncodeToString(refMac))
	return signedRef, nil
}
