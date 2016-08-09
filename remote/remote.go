package remote

import (
	"github.com/buppyio/bpy/remote/client"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
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

func GetTag(c *client.Client, name string) (string, bool, error) {
	r, err := c.TGetTag(name)
	if err != nil {
		return "", false, err
	}
	return r.Value, r.Ok, nil
}

func Tag(c *client.Client, name, value string, generation uint64) error {
	_, err := c.TTag(name, value, generation)
	return err
}

func CasTag(c *client.Client, name, oldValue, newValue string, generation uint64) (bool, error) {
	r, err := c.TCasTag(name, oldValue, newValue, generation)
	if err != nil {
		return false, err
	}
	return r.Ok, nil
}

func RemoveTag(c *client.Client, name, oldValue string, generation uint64) error {
	_, err := c.TRemoveTag(name, oldValue, generation)
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
