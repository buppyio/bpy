package remote

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/sig"
	"io"
	"io/ioutil"
	"time"
)

var (
	ErrTooSmallForEntry    = errors.New("buffer too small for stat entry")
	ErrBadReadOffset       = errors.New("bad read offset")
	ErrCorruptPackListing  = errors.New("corrupt pack listing")
	ErrCorruptRefListing   = errors.New("corrupt ref listing")
	ErrRootSignatureFailed = errors.New("root signature failed! corruption or tampering detected!")
)

type PackListing struct {
	Name string
	Size uint64
	Date time.Time
}

func ListPacks(c *client.Client) ([]PackListing, error) {
	f, err := c.Open("packs")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	listing := []PackListing{}
	for len(data) != 0 {
		if len(data) < 2 {
			return nil, ErrCorruptPackListing
		}
		namesz := int(binary.BigEndian.Uint16(data[0:2]))
		if len(data) < namesz+18 {
			return nil, ErrCorruptPackListing
		}
		listing = append(listing, PackListing{
			Name: string(data[2 : 2+namesz]),
			Size: binary.BigEndian.Uint64(data[2+namesz : 10+namesz]),
			Date: time.Unix(int64(binary.BigEndian.Uint64(data[10+namesz:18+namesz])), 0),
		})
		data = data[18+namesz:]
	}
	return listing, nil
}

func GetRoot(c *client.Client, k *bpy.Key) ([32]byte, uint64, bool, error) {
	r, err := c.TGetRoot()
	if err != nil {
		return [32]byte{}, 0, false, err
	}

	if !r.Ok {
		return [32]byte{}, 0, false, nil
	}
	signature := sig.SignValue(k, r.Value, r.Version)
	if err != nil {
		return [32]byte{}, 0, false, err
	}
	if signature != r.Signature {
		return [32]byte{}, 0, false, ErrRootSignatureFailed
	}
	h, err := bpy.ParseHash(r.Value)
	if err != nil {
		return [32]byte{}, 0, false, err
	}
	return h, r.Version, true, nil
}

func CasRoot(c *client.Client, k *bpy.Key, newHash [32]byte, newVersion uint64, generation uint64) (bool, error) {
	newValue := hex.EncodeToString(newHash[:])
	newSignature := sig.SignValue(k, newValue, newVersion)

	r, err := c.TCasRoot(newValue, newVersion, newSignature, generation)
	if err != nil {
		return false, err
	}

	return r.Ok, nil
}

func Remove(c *client.Client, path string, gcGeneration uint64) error {
	_, err := c.TRemove(path, gcGeneration)
	return err
}

func GetGeneration(c *client.Client) (uint64, error) {
	r, err := c.TGetGeneration()
	if err != nil {
		return 0, err
	}
	return r.Generation, nil
}

func StartGC(c *client.Client) (string, error) {
	idBytes := [64]byte{}
	_, err := io.ReadFull(rand.Reader, idBytes[:])
	if err != nil {
		return "", err
	}
	id := hex.EncodeToString(idBytes[:])
	_, err = c.TStartGC(id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func StopGC(c *client.Client) error {
	_, err := c.TStopGC()
	return err
}
