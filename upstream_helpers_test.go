// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Helper types used by the upstream tests, from encoding/json/encode_test.go.

package gojson

// Ref has Marshaler and Unmarshaler methods with pointer receiver.
type Ref int

func (*Ref) MarshalJSON() ([]byte, error) { return []byte(`"ref"`), nil }

func (r *Ref) UnmarshalJSON([]byte) error {
	*r = 12
	return nil
}

// RefText has Marshaler and Unmarshaler methods with pointer receiver.
type RefText int

func (*RefText) MarshalText() ([]byte, error) { return []byte(`"ref"`), nil }

func (r *RefText) UnmarshalText([]byte) error {
	*r = 13
	return nil
}

// C implements Marshaler and returns unescaped JSON.
type C int

func (C) MarshalJSON() ([]byte, error) { return []byte(`"<&>"`), nil }

// CText implements Marshaler and returns unescaped text.
type CText int

func (CText) MarshalText() ([]byte, error) { return []byte(`"<&>"`), nil }
