package remote

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote/client"
	"io"
	"io/ioutil"
	"time"
)

var (
	ErrTooSmallForEntry   = errors.New("buffer too small for stat entry")
	ErrBadReadOffset      = errors.New("bad read offset")
	ErrCorruptPackListing = errors.New("corrupt pack listing")
	ErrCorruptRefListing  = errors.New("corrupt ref listing")
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

func ListRefs(c *client.Client) ([]string, error) {
	f, err := c.Open("refs")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	listing := []string{}
	for len(data) != 0 {
		if len(data) < 2 {
			return nil, ErrCorruptPackListing
		}
		namesz := int(binary.BigEndian.Uint16(data[0:2]))
		if len(data) < namesz+2 {
			return nil, ErrCorruptRefListing
		}
		listing = append(listing, string(data[2:2+namesz]))
		data = data[2+namesz:]
	}
	return listing, nil
}

func GetRef(c *client.Client, k *bpy.Key, name string) (refs.Ref, bool, error) {
	r, err := c.TGetRef(name)
	if err != nil {
		return refs.Ref{}, false, err
	}
	if !r.Ok {
		return refs.Ref{}, false, nil
	}
	ref, err := refs.ParseRef(k, r.Value)
	return ref, true, err
}

func NewRef(c *client.Client, k *bpy.Key, name string, ref refs.Ref, generation uint64) error {
	value, err := refs.SerializeAndSign(k, ref)
	if err != nil {
		return err
	}
	_, err = c.TRef(name, value, generation)
	return err
}

func CasRef(c *client.Client, k *bpy.Key, name string, oldRef, newRef refs.Ref, generation uint64) (bool, error) {
	oldValue, err := refs.SerializeAndSign(k, oldRef)
	if err != nil {
		return false, err
	}
	newValue, err := refs.SerializeAndSign(k, newRef)
	if err != nil {
		return false, err
	}
	r, err := c.TCasRef(name, oldValue, newValue, generation)
	if err != nil {
		return false, err
	}
	return r.Ok, nil
}

func RemoveRef(c *client.Client, k *bpy.Key, name string, ref refs.Ref, generation uint64) error {
	oldValue, err := refs.SerializeAndSign(k, ref)
	if err != nil {
		return err
	}
	_, err = c.TRemoveRef(name, oldValue, generation)
	return err
}

func Remove(c *client.Client, path, gcId string) error {
	_, err := c.TRemove(path, gcId)
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
