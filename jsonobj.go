// Copyright (c) 2018 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a GNU GPLv3 license found in the
// acompanying "LICENSE" file.

// package jo implements an intermediate JSON object.
package jsonobj

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// ErrJSON is this package's base error.
type ErrJSON struct {
	ErrorString string
}

// Error implements the Error interface.
func (err ErrJSON) Error() string {
	return err.ErrorString
}

var (
	// ErrInvalidIn is returned by the Set method when the in parameter is
	// invalid, most likely nil.
	ErrInvalidIn = &ErrJSON{"invalid in value"}

	// ErrInvalidOut is returned by Get method when the specified out
	// variable is not a pointer to a variable of the type compatible with
	// the kind of value returned by Get.
	ErrInvalidOut = &ErrJSON{"invalid out type"}

	// ErrInvalidPath is returned by Get/Set methods when the specified path
	// of the JSON element is malformed.
	ErrInvalidPath = &ErrJSON{"invalid path"}

	// ErrNotFound is returned when a path form is ok but the element under
	// that path does not exist.
	ErrNotFound = &ErrJSON{"element not found"}

	// ErrOutOfRange is returned when addressing out of range Array element.
	ErrOutOfRange = &ErrJSON{"index out of range"}

	// ErrTruncate is returned when a value was successfully assigned to out
	// but the output variable was truncated or overflowed as a result of the
	// typecast.
	ErrTruncate = &ErrJSON{"json value truncated or overflowed in output"}

	// ErrTypeMissmatch is returned on unmatched in and out parameter types.
	ErrTypeMissmatch = &ErrJSON{"in out type missmatch"}
)

// JSON is an intermediate type for reading/writing values to/from a JSON
// without having to first define a type for its' structure. It's convenient
// at the price of speed. Extensive type checking in the internal callchain
// makes it somewhat slower than the usual way of addressing a field from a
// struct unmarshaled with the json package. JSON is designed so that it
// should never panic.
//
// You access JSON fields with the Get/Set methods specifying a path of the
// element in the form similar to jQuery where the syntax is as follows:
//
// Elements are addressed hierarchically in a dot notation where dot separates
// element names. Array indexes are specified via an indice in a square
// bracket as a suffix to an element name. So for example for a JSON Object
// with a property named "planets" that is an array of Objects with a String
// value "name" and a Number value "moons" to get the name of the first entry
// you'd write:
// 	jf.Get("planets[0].name", &myVar).
// To get 42nd Object from some JSON containing an array of objects:
// 	jf.Get("[42]", &myVar).
// Same rules apply to Set method.
type JSON struct {
	intf interface{} // iface is the unmarshaled JSON object.
}

// Unmarshal constructs a new JSON object from a slice of bytes.
// Returns a nil JSON and an error if one occured, *JSON otherwise.
func Unmarshal(b []byte) (*JSON, error) {
	p := &JSON{}
	if err := json.Unmarshal(b, &p.intf); err != nil {
		return nil, err
	}
	return p, nil
}

// find looks for a child element in the JSON using the specified path and
// returns the key Value with the applicable type that adresses it in its
// container (be it map or slice), the value itself it as an interface and
// a nil error on success. It returns just the error if one occured.
func (j *JSON) find(path string, parent bool) (reflect.Value, interface{}, error) {

	parentKey := reflect.ValueOf(nil)

	keys := strings.Split(path, ".")
	if len(keys) == 0 {
		return parentKey, nil, ErrInvalidPath
	}
	for _, v := range keys {
		if v == "" {
			return parentKey, nil, ErrInvalidPath
		}
	}

	result := j.intf // Updated through the loop and returned after it.
	var err error
	kl := len(keys)
	a, b, i := -1, -1, -1
	for keyi, keyv := range keys {

		a = strings.LastIndex(keyv, "[")
		b = strings.LastIndex(keyv, "]")

		if a >= 0 {
			if b <= a {
				return parentKey, nil, ErrInvalidPath
			}
			i, err = strconv.Atoi(keyv[a+1 : b])
			if err != nil {
				return parentKey, nil, ErrInvalidPath
			}
			keyv = keyv[:a]
		}

		if keyv == "" {
			si, ok := result.([]interface{})
			if !ok {
				return parentKey, nil, ErrNotFound
			}
			if i < 0 || i >= len(si) {
				return parentKey, nil, ErrOutOfRange
			}
			if parent && keyi == kl-1 {
				parentKey = reflect.ValueOf(i)
				result = si
			} else {
				result = si[i]
			}
			i = -1
		} else {
			mi, ok := result.(map[string]interface{})
			if !ok {
				return parentKey, nil, ErrNotFound
			}
			if i < 0 && parent && keyi == kl-1 {
				parentKey = reflect.ValueOf(keyv)
				result = mi
				break
			}
			iv, ok := mi[keyv]
			if !ok {
				return parentKey, nil, ErrNotFound
			}
			result = iv
			if i > -1 {
				sv, ok := iv.([]interface{})
				if !ok {
					return parentKey, nil, ErrNotFound
				}
				if i >= len(sv) {
					return parentKey, nil, ErrOutOfRange
				}
				if parent && keyi == kl-1 {
					parentKey = reflect.ValueOf(i)
					result = sv
					break
				}
				result = sv[i]
			}
		}
		i = -1
	}

	return parentKey, result, nil
}

// assign recursively assigns in to out in a manner defined by this JSON type.
func (j *JSON) assign(in, out reflect.Value) error {

	if !in.IsValid() {
		return nil
	}

	switch out.Kind() {

	case reflect.Slice:
		sl := reflect.MakeSlice(out.Type(), in.Len(), in.Len())
		for i := 0; i < in.Len(); i++ {
			if err := j.assign(in.Index(i), sl.Index(i)); err != nil {
				return err
			}
		}
		out.Set(sl)

	case reflect.Struct:
		keys := in.MapKeys()
		match := false
		for i := 0; i < out.NumField(); i++ {

			if !out.Field(i).CanSet() {
				continue
			}
			fld := out.Type().Field(i)
			tags := strings.Split(fld.Tag.Get("json"), ",")

			for k := 0; k < len(keys); k++ {
				if len(tags) > 0 {
					match = keys[k].String() == tags[0]
				} else {
					match = strings.EqualFold(keys[k].String(), fld.Name)
				}
				if !match {
					continue
				}
				val := in.MapIndex(keys[k])
				if err := j.assign(val, out.Field(i)); err != nil {
					return err
				}
				break
			}
		}

	// Booleans, Strings and Numbers are directly
	// assigned as json package defines.

	case reflect.Bool:
		v, ok := in.Interface().(bool)
		if !ok {
			return ErrInvalidOut
		}
		out.Set(reflect.ValueOf(v))
	case reflect.String:
		v, ok := in.Interface().(string)
		if !ok {
			return ErrInvalidOut
		}
		out.Set(reflect.ValueOf(v))
	case reflect.Float64:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		out.Set(reflect.ValueOf(v))

	// The rest casts the value to type of the output
	// variable and checks for rounding errors.

	case reflect.Float32:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(float32(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(float32(v)))
	case reflect.Int:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(int(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(int(v)))
	case reflect.Int8:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(int8(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(int8(v)))
	case reflect.Int16:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(int16(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(int16(v)))
	case reflect.Int32:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(int32(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(int32(v)))
	case reflect.Int64:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(int64(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(int64(v)))
	case reflect.Uint:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(uint(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(uint(v)))
	case reflect.Uint8:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(uint8(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(uint8(v)))
	case reflect.Uint16:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(uint16(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(uint16(v)))
	case reflect.Uint32:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(uint32(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(uint32(v)))
	case reflect.Uint64:
		v, ok := in.Interface().(float64)
		if !ok {
			return ErrInvalidOut
		}
		if v-float64(uint64(v)) != 0 {
			return ErrTruncate
		}
		out.Set(reflect.ValueOf(uint64(v)))
	}

	return nil
}

// Get gets a JSON value by path and writes it to out. If path is malformed
// returns ErrInvalidPath. If path specifies a non-existent element returns
// ErrNotFound. If out is not a pointer to a variable of a type compatible
// with the specified element value returns ErrInvalidTarget.
//
// It will try to assign a Number element into any type of numeric type
// including ints, uints floats and custom types with basic numeral base
// types. If the value can be assigned to out but causes the value to
// overflow or truncate function returns ErrTruncate.
//
// When assigning an Object to a struct only fields present in both input and
// settable output are assigned. Field names are matched first by tag
// respecting case, then by field name ignoring case. In other words, just
// like json package. Non-matched fields are silently skipped, meaning, you
// could end up with an empty struct without any errors.
//
// On success function returns nil.
func (j *JSON) Get(path string, out interface{}) error {

	outv := reflect.ValueOf(out)
	if !outv.IsValid() || outv.Kind() != reflect.Ptr {
		return ErrInvalidOut
	}
	outv = outv.Elem()

	_, ifc, err := j.find(path, false)
	if err != nil {
		return err
	}
	inv := reflect.ValueOf(ifc)

	return j.assign(inv, outv)
}

// Set sets a JSON element value by path. If path is malformed returns
// ErrInvalidPath. Set forces the full path of an element and the element
// itself discarding any overwritten entries without notice.
// On success function returns nil.
func (j *JSON) Set(path string, in interface{}) error {

	inv := reflect.ValueOf(in)
	if !inv.IsValid() {
		return ErrInvalidIn
	}

	key, tgt, err := j.find(path, true)
	if err != nil {
		return err
	}

	buff := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buff)
	if err := enc.Encode(in); err != nil {
		return err
	}
	var ifc interface{}
	if err := json.Unmarshal(buff.Bytes(), &ifc); err != nil {
		return err
	}

	tgtval := reflect.ValueOf(tgt)
	switch tgtval.Kind() {
	case reflect.Map:
		tgtval.SetMapIndex(key, reflect.ValueOf(ifc))
	case reflect.Slice:
		tgtval.Index(int(key.Int())).Set(reflect.ValueOf(ifc))
	default:
		panic("this shouldn't happen: parent value not map or slice")
	}

	return nil
}

// Len returns the length of the Array specified by path. If path is malformed
// returns ErrInvalidPath. If Array is not found returns ErrNotFound. Returns
// the Array length on success or -1 and an error otherwise.
func (j *JSON) Len(path string) (int, error) {

	_, slc, err := j.find(path, false)
	if err != nil {
		return -1, err
	}
	slcv, ok := slc.([]interface{})
	if !ok {
		return -1, ErrInvalidPath
	}
	return len(slcv), nil
}

// Export exports the JSON in its' current state as a slice of bytes.
func (j *JSON) Export(indent string) ([]byte, error) {

	var b []byte
	var err error
	if indent != "" {
		b, err = json.MarshalIndent(j.intf, "", indent)
	} else {
		b, err = json.Marshal(j.intf)
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}
