/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package value

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"sigs.k8s.io/structured-merge-diff/v4/internal/builder"
)

// Field is an individual key-value pair.
type Field struct {
	Name  string
	Value Value
}

// FieldList is a list of key-value pairs. Each field is expected to
// have a different name.
type FieldList []Field

// FieldListFromJSON is a helper function for reading a JSON document.
func FieldListFromJSON(input []byte) (FieldList, error) {
	parser := builder.NewFastObjParser(input)

	var fields FieldList
	for {
		rawKey, err := parser.Parse()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parsing JSON: %v", err)
		}

		rawValue, err := parser.Parse()
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected EOF")
		} else if err != nil {
			return nil, fmt.Errorf("parsing JSON: %v", err)
		}

		k, err := builder.UnmarshalString(rawKey)
		if err != nil {
			return nil, fmt.Errorf("parsing JSON: %v", err)
		}

		v, err := builder.UnmarshalInterface(rawValue)
		if err != nil {
			return nil, fmt.Errorf("parsing JSON: %v", err)
		}

		fields = append(fields, Field{Name: k, Value: NewValueInterface(v)})
	}

	return fields, nil
}

// FieldListToJSON is a helper function for producing a JSON document.
func FieldListToJSON(v FieldList, w *builder.JSONBuilder) error {
	w.WriteByte('{')
	for i, f := range v {
		if err := w.WriteJSON(f.Name); err != nil {
			return err
		}
		w.WriteByte(':')
		if err := w.WriteJSON(f.Value.Unstructured()); err != nil {
			return err
		}
		if i < len(v)-1 {
			w.WriteByte(',')
		}
	}
	w.WriteByte('}')
	return nil
}

// Sort sorts the field list by Name.
func (f FieldList) Sort() {
	if len(f) < 2 {
		return
	}
	if len(f) == 2 {
		if f[1].Name < f[0].Name {
			f[0], f[1] = f[1], f[0]
		}
		return
	}
	sort.SliceStable(f, func(i, j int) bool {
		return f[i].Name < f[j].Name
	})
}

// Less compares two lists lexically.
func (f FieldList) Less(rhs FieldList) bool {
	return f.Compare(rhs) == -1
}

// Compare compares two lists lexically. The result will be 0 if f==rhs, -1
// if f < rhs, and +1 if f > rhs.
func (f FieldList) Compare(rhs FieldList) int {
	i := 0
	for {
		if i >= len(f) && i >= len(rhs) {
			// Maps are the same length and all items are equal.
			return 0
		}
		if i >= len(f) {
			// F is shorter.
			return -1
		}
		if i >= len(rhs) {
			// RHS is shorter.
			return 1
		}
		if c := strings.Compare(f[i].Name, rhs[i].Name); c != 0 {
			return c
		}
		if c := Compare(f[i].Value, rhs[i].Value); c != 0 {
			return c
		}
		// The items are equal; continue.
		i++
	}
}

// Equals returns true if the two fieldslist are equals, false otherwise.
func (f FieldList) Equals(rhs FieldList) bool {
	if len(f) != len(rhs) {
		return false
	}
	for i := range f {
		if f[i].Name != rhs[i].Name {
			return false
		}
		if !Equals(f[i].Value, rhs[i].Value) {
			return false
		}
	}
	return true
}
