package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var deploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: the-deployment
spec:
  replicas: 3
  template:
    metadata:
      name: the-pod
    spec:
      containers:
      - name: the-container
        image: nginx

`

var overlayYAML = `
# v1.PodTemplateSpec
template:
  metadata:
    name: the-pod
  spec:
    containers:
    - name: the-container-remix
      image: the-matrix
`

type overlay struct {
	key      string
	metadata map[string]string
	data     map[interface{}]interface{}
	found    bool
}

func main() {
	if err := apply([]byte(deploymentYAML), []byte(overlayYAML)); err != nil {
		logrus.Fatal(err)
	}
}

func apply(baseYAML, overlayYAML []byte) error {
	base := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(baseYAML, &base); err != nil {
		return errors.Wrap(err, "reading base yaml")
	}

	o, err := newOverlay(overlayYAML)
	if err != nil {
		return errors.Wrap(err, "generating overlay")
	}
	fmt.Printf("%+v", o)

	overlayRecursive(base, o)

	out, err := yaml.Marshal(base)
	if err != nil {
		return errors.Wrap(err, "marshaling base yaml")
	}
	fmt.Println(string(out))

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
				o.metadata[mk.(string)] = mv.(string)
			}
			return o, nil
		}
	}

	return nil, nil
}

func overlayRecursive(i interface{}, o *overlay) {
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
			t[k] = o.data
			o.found = true
			fmt.Println("found")
		}
	}
}

func matches(baseKey string, baseValue interface{}, o *overlay) bool {
	// fmt.Printf("matches: baseKey: %s baseValue:%+v overlay: %+v\n", baseKey, baseValue, o)
	if baseKey == "metadata" {
		baseMeta, ok := convert(baseValue)
		if ok {
			for _, k := range []string{"name", "generateName", "namespace"} {
				if baseMeta[k] != o.metadata[k] {
					return false
				}
			}
			return true
		}
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
			return nil, false
		}
		vStr, ok := v.(string)
		if !ok {
			return nil, false
		}
		ret[kStr] = vStr
	}
	return ret, true
}
