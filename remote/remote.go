package remote

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/remote/client"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"time"
)

var (
	ErrTooSmallForEntry   = errors.New("buffer too small for stat entry")
	ErrBadReadOffset      = errors.New("bad read offset")
	ErrCorruptPackListing = errors.New("corrupt pack listing")
	ErrCorruptTagListing  = errors.New("corrupt tag listing")
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

func ListTags(c *client.Client) ([]string, error) {
	f, err := c.Open("tags")
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
			return nil, ErrCorruptTagListing
		}
		listing = append(listing, string(data[2:2+namesz]))
		data = data[2+namesz:]
	}
	return listing, nil
}

func Tag(c *client.Client, name string, hash [32]byte) error {
	_, err := c.TTag(name, hex.EncodeToString(hash[:]))
	return err
}

func GetTag(c *client.Client, name string) ([32]byte, error) {
	r, err := c.TGetTag(name)
	if err != nil {
		return [32]byte{}, err
	}
	return bpy.ParseHash(r.Value)
}

func RemoveTag(c *client.Client, name string) error {
	_, err := c.TRemoveTag(name)
	return err
}