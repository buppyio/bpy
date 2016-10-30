package sig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/buppyio/bpy"
)

func SignValue(k *bpy.Key, value string, version uint64) string {
	toSign := fmt.Sprintf("%s:%s", value, version)
	mac := hmac.New(sha256.New, k.HmacKey[:])
	mac.Write([]byte(toSign))
	hashMac := mac.Sum(nil)
	sig := hex.EncodeToString(hashMac)
	return sig
}
