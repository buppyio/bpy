package remote

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/sig"
	"io/ioutil"
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
		if len(data) < namesz+10 {
			return nil, ErrCorruptPackListing
		}
		listing = append(listing, PackListing{
			Name: string(data[2 : 2+namesz]),
			Size: binary.BigEndian.Uint64(data[2+namesz : 10+namesz]),
		})
		data = data[10+namesz:]
	}
	return listing, nil
}

func GetRoot(c *client.Client, k *bpy.Key) ([32]byte, string, bool, error) {
	r, err := c.TGetRoot()
	if err != nil {
		return [32]byte{}, "", false, err
	}

	if !r.Ok {
		return [32]byte{}, r.Version, false, nil
	}
	signature := sig.SignValue(k, r.Value, r.Version)
	if err != nil {
		return [32]byte{}, "", false, err
	}
	if signature != r.Signature {
		return [32]byte{}, "", false, ErrRootSignatureFailed
	}
	h, err := bpy.ParseHash(r.Value)
	if err != nil {
		return [32]byte{}, "", false, err
	}
	return h, r.Version, true, nil
}

func CasRoot(c *client.Client, k *bpy.Key, newHash [32]byte, newVersion, epoch string) (bool, error) {
	newValue := hex.EncodeToString(newHash[:])
	newSignature := sig.SignValue(k, newValue, newVersion)

	r, err := c.TCasRoot(newValue, newVersion, newSignature, epoch)
	if err != nil {
		return false, err
	}

	return r.Ok, nil
}

func Remove(c *client.Client, path, epoch string) error {
	_, err := c.TRemove(path, epoch)
	return err
}

func GetEpoch(c *client.Client) (string, error) {
	r, err := c.TGetEpoch()
	if err != nil {
		return "", err
	}
	return r.Epoch, nil
}

func StartGC(c *client.Client) (string, error) {
	r, err := c.TStartGC()
	if err != nil {
		return "", err
	}
	return r.Epoch, nil
}

func StopGC(c *client.Client) error {
	_, err := c.TStopGC()
	return err
}
