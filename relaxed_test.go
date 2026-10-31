package gojson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
)

type point struct {
	X, Y float64
	Name string `json:"name"`
}

func TestRelaxedUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		in   string
		want any // as decoded into any
	}{
		{"trailing comma array", AllowTrailingCommas, `[1, 2, 3,]`, []any{1.0, 2.0, 3.0}},
		{"trailing comma object", AllowTrailingCommas, `{"a": 1, "b": 2,}`, map[string]any{"a": 1.0, "b": 2.0}},
		{"trailing comma nested", AllowTrailingCommas, `[[1,],{"a":[],},]`, []any{[]any{1.0}, map[string]any{"a": []any{}}}},
		{"trailing comma whitespace", AllowTrailingCommas, "[1 ,\n\t]", []any{1.0}},
		{"unquoted keys", AllowUnquotedKeys, `{a: 1, _b$2: 2, $: 3, "q": 4}`, map[string]any{"a": 1.0, "_b$2": 2.0, "$": 3.0, "q": 4.0}},
		{"unquoted keyword keys", AllowUnquotedKeys, `{true: 1, null: 2, NaN: 3}`, map[string]any{"true": 1.0, "null": 2.0, "NaN": 3.0}},
		{"unquoted key spacing", AllowUnquotedKeys, `{ a :1}`, map[string]any{"a": 1.0}},
		{"line comment", AllowComments, "// head\n[1, // one\n 2]// tail", []any{1.0, 2.0}},
		{"block comment", AllowComments, `/* a */[/**/1/* b */,/***/2/* c **/]/*x*/`, []any{1.0, 2.0}},
		{"comment in object", AllowComments, `{/*k*/"a"/*c*/:/*v*/1/*e*/}`, map[string]any{"a": 1.0}},
		{"comment-like string", AllowComments, `"/* not a comment */ // nor this"`, "/* not a comment */ // nor this"},
		{"line comment at EOF", AllowComments, "1 // no newline", 1.0},
		{"comment after number", AllowComments, "[1//x\n]", []any{1.0}},
		{"comment before trailing", Relaxed, "[1, /* c */ ]", []any{1.0}},
		{"all", Relaxed, `
			// config
			{
				name: "x", /* inline */
				list: [1, 2, 3,],
				nested: {deep: true,},
			}`, map[string]any{"name": "x", "list": []any{1.0, 2.0, 3.0}, "nested": map[string]any{"deep": true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got any
			if err := UnmarshalWith([]byte(tt.in), &got, tt.opts); err != nil {
				t.Fatalf("UnmarshalWith error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
			// Each extension must be rejected by default, like encoding/json.
			if err := Unmarshal([]byte(tt.in), new(any)); err == nil && !strings.Contains(tt.name, "string") {
				t.Errorf("strict Unmarshal accepted %q", tt.in)
			}
			if !ValidWith([]byte(tt.in), tt.opts) {
				t.Errorf("ValidWith = false")
			}
			// Standardize must produce input that encoding/json decodes the same way.
			std, err := Standardize([]byte(tt.in), tt.opts)
			if err != nil {
				t.Fatalf("Standardize error: %v", err)
			}
			var stdGot any
			if err := json.Unmarshal(std, &stdGot); err != nil {
				t.Fatalf("json.Unmarshal(Standardize) error: %v\n%s", err, std)
			}
			if !reflect.DeepEqual(stdGot, tt.want) {
				t.Fatalf("standardized got %#v, want %#v", stdGot, tt.want)
			}
		})
	}
}

func TestNonFinite(t *testing.T) {
	var v struct {
		A, B, C, D float64
		E          float32
		N          Number
		I          any
	}
	in := `{"A": NaN, "B": Infinity, "C": +Infinity, "D": -Infinity, "E": -Infinity, "N": NaN, "I": -Infinity}`
	if err := Unmarshal([]byte(in), &v); err == nil {
		t.Fatal("strict Unmarshal accepted NaN")
	}
	if err := UnmarshalWith([]byte(in), &v, AllowNonFinite); err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(v.A) || !math.IsInf(v.B, 1) || !math.IsInf(v.C, 1) || !math.IsInf(v.D, -1) || !math.IsInf(float64(v.E), -1) {
		t.Errorf("got %+v", v)
	}
	if v.N != "NaN" {
		t.Errorf("Number = %q, want NaN", v.N)
	}
	if f, err := v.N.Float64(); err != nil || !math.IsNaN(f) {
		t.Errorf("Number.Float64() = %v, %v", f, err)
	}
	if f, ok := v.I.(float64); !ok || !math.IsInf(f, -1) {
		t.Errorf("interface = %#v", v.I)
	}

	// Into integers it's a type error, like an out-of-range number.
	var n struct{ X int }
	err := UnmarshalWith([]byte(`{"X": NaN}`), &n, AllowNonFinite)
	var ute *UnmarshalTypeError
	if !errors.As(err, &ute) || ute.Value != "number NaN" || ute.Field != "X" {
		t.Errorf("into int: err = %v", err)
	}

	// UseNumber keeps the literal text.
	dec := NewDecoder(strings.NewReader(`[NaN, -Infinity, 1]`))
	dec.SetOptions(AllowNonFinite)
	dec.UseNumber()
	var arr []any
	if err := dec.Decode(&arr); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(arr, []any{Number("NaN"), Number("-Infinity"), Number("1")}) {
		t.Errorf("UseNumber: got %#v", arr)
	}

	// ,string fields.
	var q struct {
		F float64 `json:",string"`
	}
	if err := UnmarshalWith([]byte(`{"F": "Infinity"}`), &q, AllowNonFinite); err != nil || !math.IsInf(q.F, 1) {
		t.Errorf(",string: %v %v", q.F, err)
	}

	// Standardize can't represent them.
	if _, err := Standardize([]byte(`[1, NaN]`), AllowNonFinite); err == nil || !strings.Contains(err.Error(), "NaN") {
		t.Errorf("Standardize(NaN) err = %v", err)
	}
}

func TestStruct(t *testing.T) {
	in := `{
		// case-insensitive matching, tags etc. work as in encoding/json
		x: 1.5, y: -Infinity,
		name: "p",
	}`
	var p point
	if err := UnmarshalWith([]byte(in), &p, Relaxed); err != nil {
		t.Fatal(err)
	}
	if p.X != 1.5 || !math.IsInf(p.Y, -1) || p.Name != "p" {
		t.Errorf("got %+v", p)
	}

	var m map[int]string
	if err := UnmarshalWith([]byte(`{1: "x"}`), &m, Relaxed); err == nil {
		t.Errorf("numeric unquoted key accepted")
	}
}

type rawCapture struct{ raw string }

func (r *rawCapture) UnmarshalJSON(b []byte) error {
	r.raw = string(b)
	return nil
}

type textKey string

func (k *textKey) UnmarshalText(b []byte) error {
	*k = textKey("<" + string(b) + ">")
	return nil
}

func TestUnmarshalerGetsStandardJSON(t *testing.T) {
	in := `{v: {a: [1, 2,], /* c */ "b": NaN,}, r: [x, 1]}`
	var v struct {
		V rawCapture
		R RawMessage
	}
	// "x" isn't a valid value; make the input valid first.
	in = strings.Replace(in, "[x, 1]", "[true, 1 // c\n,]", 1)
	if err := UnmarshalWith([]byte(in), &v, Relaxed); err != nil {
		t.Fatal(err)
	}
	if want := `{"a": [1, 2],  "b": NaN}`; v.V.raw != want {
		t.Errorf("Unmarshaler got %q, want %q", v.V.raw, want)
	}
	if want := "[true, 1 ]"; string(v.R) != want {
		t.Errorf("RawMessage got %q, want %q", v.R, want)
	}
	if !json.Valid(v.R) {
		t.Errorf("RawMessage is not standard JSON")
	}

	var m map[textKey]int
	if err := UnmarshalWith([]byte(`{abc: 1, "d": 2}`), &m, Relaxed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, map[textKey]int{"<abc>": 1, "<d>": 2}) {
		t.Errorf("TextUnmarshaler keys: %v", m)
	}
}

func TestRelaxedSyntaxErrors(t *testing.T) {
	tests := []struct {
		opts   Options
		in     string
		msg    string
		offset int64
	}{
		{0, `[1,]`, "invalid character ']' looking for beginning of value", 4},
		{0, `{"a":1,}`, "invalid character '}' looking for beginning of object key string", 8},
		{0, `{a:1}`, "invalid character 'a' looking for beginning of object key string", 2},
		{0, `NaN`, "invalid character 'N' looking for beginning of value", 1},
		{0, `[1] // x`, "invalid character '/' after top-level value", 5},
		{Relaxed, `[,]`, "invalid character ',' looking for beginning of value", 2},
		{Relaxed, `[1,,]`, "invalid character ',' looking for beginning of value", 4},
		{Relaxed, `{,}`, "invalid character ',' looking for beginning of object key string", 2},
		{Relaxed, `{1a: 2}`, "invalid character '1' looking for beginning of object key string", 2},
		{Relaxed, `{a-b: 2}`, "invalid character '-' after object key", 3},
		{Relaxed, `[NaX]`, "invalid character 'X' in literal NaN (expecting 'N')", 4},
		// Same message as encoding/json gives for a truncated "tru".
		{Relaxed, `-Inf`, "invalid character ' ' in literal -Infinity (expecting 'i')", 4},
		{Relaxed, `+1`, "invalid character '1' in literal +Infinity (expecting 'I')", 2},
		{Relaxed, `nan`, "invalid character 'a' in literal null (expecting 'u')", 2},
		{Relaxed, `[1 /x]`, "invalid character 'x' after '/' (expecting '/' or '*' to begin comment)", 5},
		{Relaxed, `[1] /* x`, "unexpected end of JSON input", 8},
		{Relaxed, `[1] /`, "unexpected end of JSON input", 5},
		{Relaxed, `// only a comment`, "unexpected end of JSON input", 17},
		{Relaxed, `{a}`, "invalid character '}' after object key", 3},
		{Relaxed, `{"a": b}`, "invalid character 'b' looking for beginning of value", 7},
	}
	for _, tt := range tests {
		err := UnmarshalWith([]byte(tt.in), new(any), tt.opts)
		var se *SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("%q: err = %v, want SyntaxError", tt.in, err)
			continue
		}
		if se.Error() != tt.msg || se.Offset != tt.offset {
			t.Errorf("%q: got %q at %d, want %q at %d", tt.in, se.Error(), se.Offset, tt.msg, tt.offset)
		}
		if ValidWith([]byte(tt.in), tt.opts) {
			t.Errorf("%q: ValidWith = true", tt.in)
		}
	}
}

func TestDecoderStream(t *testing.T) {
	in := `// leading comment
	{a: 1,} /* between */ [2,]
	// another
	NaN 3 // trailing`
	dec := NewDecoder(strings.NewReader(in))
	dec.SetOptions(Relaxed)
	var got []any
	for {
		var v any
		err := dec.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if len(got) != 4 || !reflect.DeepEqual(got[0], map[string]any{"a": 1.0}) ||
		!reflect.DeepEqual(got[1], []any{2.0}) || !math.IsNaN(got[2].(float64)) || got[3] != 3.0 {
		t.Errorf("got %#v", got)
	}

	// Unterminated comment at EOF.
	dec = NewDecoder(strings.NewReader("1 /* never closed"))
	dec.SetOptions(Relaxed)
	if err := dec.Decode(new(any)); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(new(any)); err != io.ErrUnexpectedEOF {
		t.Errorf("err = %v, want ErrUnexpectedEOF", err)
	}
}

// oneByteReader makes the Decoder refill its buffer at every byte,
// exercising comments and identifiers that span reads.
type oneByteReader struct{ r io.Reader }

func (o oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return o.r.Read(p[:1])
}

func TestDecoderToken(t *testing.T) {
	in := `/* c */ {key: [1, NaN, /* c */ ], "q": {x: true,}, // end
	}`
	for _, r := range []func() io.Reader{
		func() io.Reader { return strings.NewReader(in) },
		func() io.Reader { return oneByteReader{strings.NewReader(in)} },
	} {
		dec := NewDecoder(r())
		dec.SetOptions(Relaxed)
		var toks []string
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Token error after %v: %v", toks, err)
			}
			toks = append(toks, strings.ReplaceAll(strings.TrimSpace(
				strings.ReplaceAll(strings.Trim(stringOf(tok), " "), "\n", "")), "NaN", "nan"))
		}
		want := "{ key [ 1 nan ] q { x true } }"
		if got := strings.Join(toks, " "); got != want {
			t.Errorf("tokens = %q, want %q", got, want)
		}
	}

	// More() must skip comments too.
	dec := NewDecoder(strings.NewReader(`[1 /* c */ ]`))
	dec.SetOptions(Relaxed)
	dec.Token()
	dec.Token()
	if dec.More() {
		t.Errorf("More() = true before ']'")
	}
}

func stringOf(tok Token) string {
	b := new(bytes.Buffer)
	switch tok := tok.(type) {
	case Delim:
		b.WriteString(tok.String())
	case float64:
		if math.IsNaN(tok) {
			return "NaN"
		}
		j, _ := json.Marshal(tok)
		b.Write(j)
	case string:
		b.WriteString(tok)
	default:
		j, _ := json.Marshal(tok)
		b.Write(j)
	}
	return b.String()
}

func TestRelaxedDecodeWithOneByteReader(t *testing.T) {
	in := "// c\n{a: [1, -Infinity,], /* x */ b: {c: null,},}\n/* y */"
	dec := NewDecoder(oneByteReader{strings.NewReader(in)})
	dec.SetOptions(Relaxed)
	var v map[string]any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	if a := v["a"].([]any); len(a) != 2 || !math.IsInf(a[1].(float64), -1) {
		t.Errorf("got %#v", v)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		t.Errorf("second Decode: %v, want EOF", err)
	}
}
