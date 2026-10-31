package gojson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type fuzzTarget struct {
	A  int
	B  string `json:"b,omitempty"`
	C  []float64
	D  map[string]any
	E  *fuzzTarget
	F  bool  `json:",string"`
	G  uint8 `json:"g"`
	H  any
	I  [2]int
	J  RawMessage
	K  Number
	KK map[int]string
}

func decodeBoth(t *testing.T, data []byte, newV func() any, opts Options) {
	v1, v2 := newV(), newV()
	err1 := json.Unmarshal(data, v1)
	err2 := UnmarshalWith(data, v2, opts)
	if err1 == nil {
		if err2 != nil {
			t.Fatalf("%q: encoding/json ok, gojson(%d) error: %v", data, opts, err2)
		}
	} else if err2 == nil {
		if opts == 0 {
			t.Fatalf("%q: encoding/json error %v, gojson ok", data, err1)
		}
		return // relaxed may accept more
	} else if opts == 0 && err1.Error() != err2.Error() {
		t.Fatalf("%q: error mismatch:\n  encoding/json: %v\n  gojson:        %v", data, err1, err2)
	}
	if err1 != nil && opts != 0 {
		if _, ok := err1.(*json.SyntaxError); ok {
			return // relaxed decoder legitimately sees different syntax
		}
	}
	if !same(v1, v2) {
		t.Fatalf("%q (opts %d): result mismatch:\n  encoding/json: %#v\n  gojson:        %#v", data, opts, v1, v2)
	}
}

// FuzzCompat checks that on standard JSON gojson behaves exactly like
// encoding/json, with and without extensions enabled, and that relaxed input
// decodes like its standardized form.
func FuzzCompat(f *testing.F) {
	for _, s := range []string{
		`{"A":1,"b":"x","C":[1,2.5],"D":{"x":[null]},"E":{"A":2},"F":"true","g":255,"H":[{}],"I":[1,2,3],"J":{"z":1},"K":12,"KK":{"1":"a"}}`,
		`{"a":-1e400}`, `{"G":256}`, `{"F":true}`, `[1,2]`, `"\ud800"`, `{"KK":{"x":"a"}}`,
		`{a: 1, /* c */ C: [1,2,],}`, `{"H": NaN, "C": [Infinity, -Infinity]}`, `// x` + "\n" + `{"E": {"E": null,},}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, opts := range []Options{0, Relaxed, AllowComments, AllowTrailingCommas | AllowUnquotedKeys} {
			decodeBoth(t, data, func() any { return new(any) }, opts)
			decodeBoth(t, data, func() any { return new(fuzzTarget) }, opts)
			decodeBoth(t, data, func() any { return new([]int) }, opts)
			decodeBoth(t, data, func() any { return new(map[string]fuzzTarget) }, opts)
			if Valid(data) != json.Valid(data) {
				t.Fatalf("%q: Valid mismatch", data)
			}
			if json.Valid(data) && !ValidWith(data, opts) {
				t.Fatalf("%q: valid JSON rejected with opts %d", data, opts)
			}
		}
		// Relaxed input must decode like its standardized form.
		std, err := Standardize(data, Relaxed&^AllowNonFinite)
		if err != nil {
			return
		}
		if !json.Valid(std) {
			t.Fatalf("Standardize(%q) = %q is not valid JSON", data, std)
		}
		var a, b fuzzTarget
		errA := UnmarshalWith(data, &a, Relaxed&^AllowNonFinite)
		errB := json.Unmarshal(std, &b)
		if (errA == nil) != (errB == nil) {
			t.Fatalf("%q: relaxed err %v, standardized err %v", data, errA, errB)
		}
		if errA == nil && !same(a, b) {
			t.Fatalf("%q: relaxed %#v != standardized %#v", data, a, b)
		}
	})
}

// same is reflect.DeepEqual, except that NaNs compare equal.
func same(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return fmt.Sprint(a) == fmt.Sprint(b) // NaN can't be marshaled
	}
	return string(ja) == string(jb)
}
