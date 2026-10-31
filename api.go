// Package gojson is a drop-in replacement for the decoding side of
// encoding/json that can optionally accept a few common extensions to the
// JSON syntax:
//
//   - [AllowTrailingCommas]: a comma after the last element of an array or
//     object, as in [1, 2, 3,] or {"a": 1,}.
//   - [AllowUnquotedKeys]: object keys that are identifiers
//     ([a-zA-Z_$][a-zA-Z0-9_$]*) need not be quoted, as in {name: "x"}.
//   - [AllowComments]: // line comments and /* block comments */ wherever
//     whitespace is allowed.
//   - [AllowNonFinite]: the number literals NaN, Infinity, +Infinity and
//     -Infinity.
//
// The decoder is a fork of the encoding/json (v1) decoder, so with no options
// enabled [Unmarshal], [Valid] and [Decoder] behave exactly like their
// encoding/json counterparts: same mapping rules, struct tags, Unmarshaler /
// TextUnmarshaler handling, errors and error messages. Use [UnmarshalWith],
// [ValidWith] or [Decoder.SetOptions] to enable extensions.
//
// Types shared with the standard library (Number, RawMessage,
// Unmarshaler, UnmarshalTypeError, ...) are aliases of the encoding/json
// types, so values and errors can be used interchangeably with code written
// against encoding/json. Encoding functions are forwarded to encoding/json.
//
// Values implementing [Unmarshaler] (including [RawMessage]) receive their
// input rewritten to standard JSON: comments and trailing commas are removed
// and unquoted keys are quoted. NaN and Infinity have no standard JSON
// representation and are passed through as is.
package gojson

import (
	"encoding/json"
	"io"
)

// Options is a set of JSON syntax extensions accepted by the decoder.
// The zero value means strict standard JSON, as in encoding/json.
type Options uint

const (
	// AllowTrailingCommas accepts a comma after the last element of an
	// array or the last member of an object.
	AllowTrailingCommas Options = 1 << iota

	// AllowUnquotedKeys accepts object keys written as bare identifiers
	// matching [a-zA-Z_$][a-zA-Z0-9_$]*.
	AllowUnquotedKeys

	// AllowComments accepts // and /* */ comments wherever whitespace is
	// allowed.
	AllowComments

	// AllowNonFinite accepts the literals NaN, Infinity, +Infinity and
	// -Infinity wherever a number is allowed. They decode into floats (and
	// interface values) as the corresponding IEEE 754 values; with
	// UseNumber or into a Number they are kept as their literal text,
	// which Number.Float64 understands.
	AllowNonFinite

	// Relaxed enables all extensions.
	Relaxed = AllowTrailingCommas | AllowUnquotedKeys | AllowComments | AllowNonFinite
)

// Types shared with encoding/json.
type (
	Unmarshaler           = json.Unmarshaler
	Marshaler             = json.Marshaler
	Number                = json.Number
	RawMessage            = json.RawMessage
	Delim                 = json.Delim
	Token                 = json.Token
	Encoder               = json.Encoder
	UnmarshalTypeError    = json.UnmarshalTypeError
	InvalidUnmarshalError = json.InvalidUnmarshalError
	MarshalerError        = json.MarshalerError
	UnsupportedTypeError  = json.UnsupportedTypeError
	UnsupportedValueError = json.UnsupportedValueError

	// Deprecated: No longer used; kept for compatibility.
	UnmarshalFieldError = json.UnmarshalFieldError
	// Deprecated: No longer used; kept for compatibility.
	InvalidUTF8Error = json.InvalidUTF8Error
)

// Marshal is [encoding/json.Marshal].
func Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// MarshalIndent is [encoding/json.MarshalIndent].
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// NewEncoder is [encoding/json.NewEncoder].
func NewEncoder(w io.Writer) *Encoder { return json.NewEncoder(w) }
