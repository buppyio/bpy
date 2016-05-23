package tags

import (
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
)

func List(remote *client9.Client) ([]string, error) {
	stats, err := remote.Ls("tags")
	if err != nil {
		return nil, err
	}
	tags := make([]string, len(stats), len(stats))
	for i, st := range stats {
		tags[i] = st.Name
	}
	sort.Strings(tags)
	return tags, nil
}

func Cas(remote *client9.Client, tag, oldval, newval string) error {
	f, err := remote.Open("ctl", proto9.OREAD)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Twrite(0, []byte(fmt.Sprintf("cas %s %s %s", tag, oldval, newval)))
	if err != nil {
		return err
	}
	return nil
}

func Set(remote *client9.Client, tag, val string) error {
	f, err := remote.Open("ctl", proto9.OREAD)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Twrite(0, []byte(fmt.Sprintf("set %s %s", tag, val)))
	if err != nil {
		return err
	}
	return nil
}

func Get(remote *client9.Client, tag string) (string, error) {
	f, err := remote.Open(path.Join("tags", tag), proto9.OREAD)
	if err != nil {
		return "", err
	}
	defer f.Close()
	val, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(val), nil
}
