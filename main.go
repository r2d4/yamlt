package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type overlay struct {
	key      string
	metadata map[string]string
	data     map[interface{}]interface{}
	found    bool
}

func main() {
	if len(os.Args) < 3 {
		logrus.Fatal("Need to specify at least two files, a base and an overlay")
	}
	baseYAML, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		logrus.Fatal(err, "reading base yaml file")
	}

	overlayYAML, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		logrus.Fatal(err, "reading overlay yaml file")
	}
	if err := apply(os.Stdout, []byte(baseYAML), []byte(overlayYAML)); err != nil {
		logrus.Fatal(err)
	}

}

func apply(out io.Writer, baseYAML, overlayYAML []byte) error {
	base := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(baseYAML, &base); err != nil {
		return errors.Wrap(err, "reading base yaml")
	}

	o, err := newOverlay(overlayYAML)
	if err != nil {
		return errors.Wrap(err, "generating overlay")
	}
	overlayRecursive(base, o)
	newManifest, err := yaml.Marshal(base)
	if err != nil {
		return errors.Wrap(err, "marshaling base yaml")
	}
	fmt.Fprint(out, string(newManifest))

	return nil
}

func newOverlay(overlayYAML []byte) (*overlay, error) {
	m := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(overlayYAML, &m); err != nil {
		return nil, errors.Wrap(err, "reading overlay yaml")
	}
	o := &overlay{
		data:     m,
		metadata: map[string]string{},
	}
	// Use top level key as overlay key
	for k := range m {
		o.key = k.(string)
		break
	}
	// Look for object meta.
	// Next level key should be map
	fields, ok := m[o.key].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("Overlay needs to have an metav1.ObjectMeta: %T", m[o.key])
	}
	for k, v := range fields {
		//fmt.Printf("key: %+v, value: %+v\n", k, v)
		if k.(string) == "metadata" {
			metadata, ok := v.(map[interface{}]interface{})
			if !ok {
				continue
			}
			for mk, mv := range metadata {
				mkStr, ok := mk.(string)
				if !ok {
					continue
				}
				mvStr, ok := mv.(string)
				if !ok {
					continue
				}
				o.metadata[mkStr] = mvStr
			}
			return o, nil
		}
	}

	return nil, nil
}

func overlayRecursive(i interface{}, o *overlay) {
	if o.found {
		return
	}
	switch t := i.(type) {
	case []interface{}:
		//fmt.Printf("type is %T: value: %+v\n\n", t, t)
		for _, v := range t {
			overlayRecursive(v, o)
		}
	case map[interface{}]interface{}:
		//fmt.Printf("type is %T: value: %+v\n\n", t, t)
		for k, v := range t {
			if !matches(k.(string), v, o) {
				overlayRecursive(v, o)
				continue
			}
			t[o.key] = o.data[o.key]
			o.found = true
			return
		}
	}
}

func matches(baseKey string, baseValue interface{}, o *overlay) bool {
	if baseKey != o.key {
		return false
	}
	nestedMap, ok := baseValue.(map[interface{}]interface{})
	if !ok {
		return false
	}
	metadata, ok := nestedMap["metadata"]
	if !ok {
		return false
	}
	baseMeta, ok := convert(metadata)
	if !ok {
		return false
	}
	for _, k := range []string{"name", "generateName", "namespace"} {
		if baseMeta[k] != o.metadata[k] {
			return false
		}
		return true
	}

	return false
}

func convert(i interface{}) (map[string]string, bool) {
	m, ok := i.(map[interface{}]interface{})
	if !ok {
		return nil, false
	}
	ret := map[string]string{}
	for k, v := range m {
		kStr, ok := k.(string)
		if !ok {
			continue
		}
		vStr, ok := v.(string)
		if !ok {
			continue
		}
		ret[kStr] = vStr
	}
	return ret, true
}
