package main

import (
	"fmt"
	"reflect"

	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var yaml = `
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

func main() {
	_ = metav1.ObjectMeta{}
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
	}

	deployment := obj.(*v1.Deployment)
	// spew.Dump(&deployment)

	getAllEmbededTypes(deployment)
	//fmt.Printf("%#v\n", deployment)
}

func getAllEmbededTypes(d *v1.Deployment) error {
	translate(d)
	return nil
}

func translate(obj interface{}) interface{} {
	// Wrap the original in a reflect.Value
	original := reflect.ValueOf(obj)

	copy := reflect.New(original.Type()).Elem()
	translateRecursive(original)

	// Remove the reflection wrapper
	return copy.Interface()
}

func translateRecursive(original reflect.Value) {
	//fmt.Printf("original type %T, original value %+v, original kind %s\n", original, original, original.Kind())
	switch original.Kind() {
	// The first cases handle nested structures and translate them recursively

	// If it is a pointer we need to unwrap and call once again
	case reflect.Ptr:
		// To get the actual value of the original we have to call Elem()
		// At the same time this unwraps the pointer so we don't end up in
		// an infinite recursion
		originalValue := original.Elem()
		// Check if the pointer is nil
		if !originalValue.IsValid() {
			return
		}
		// Unwrap the newly created pointer
		translateRecursive(originalValue)

	// If it is an interface (which is very similar to a pointer), do basically the
	// same as for the pointer. Though a pointer is not the same as an interface so
	// note that we have to call Elem() after creating a new object because otherwise
	// we would end up with an actual pointer
	case reflect.Interface:
		// Get rid of the wrapping interface
		originalValue := original.Elem()

		// Create a new object. Now new gives us a pointer, but we want the value it
		// points to, so we have to call Elem() to unwrap it
		translateRecursive(originalValue)

	// If it is a struct we translate each field
	case reflect.Struct:
		fmt.Printf("the type %T\n", original.Interface())
		for i := 0; i < original.NumField(); i++ {
			if original.Field(i).CanInterface() {
				typemeta, ok := original.Field(i).Interface().(metav1.ObjectMeta)
				if ok {
					fmt.Printf("\n\nFOUND A TYPE! from %T %+v\n\n", original.Interface(), typemeta)
				}
			}
			translateRecursive(original.Field(i))
		}

	// If it is a slice we create a new slice and translate each element
	case reflect.Slice:
		for i := 0; i < original.Len(); i++ {
			translateRecursive(original.Index(i))
		}

	// If it is a map we create a new map and translate each value
	case reflect.Map:
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			// New gives us a pointer, but again we want the value
			translateRecursive(originalValue)
		}

	// Otherwise we cannot traverse anywhere so this finishes the the recursion

	// If it is a string translate it (yay finally we're doing what we came for)
	case reflect.String:

	// And everything else will simply be taken from the original
	default:
	}

}
