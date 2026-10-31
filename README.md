# gojson

A drop-in replacement for the decoding side of Go's `encoding/json` that can
optionally accept a few common extensions to JSON syntax: trailing commas,
unquoted object keys, comments, and `NaN` / `Infinity`.

With no options enabled it behaves **exactly** like `encoding/json`: the
decoder is a fork of the standard library's (v1) decoder, and Go's own
upstream decoder, scanner and stream tests pass against it unchanged.

```go
import json "github.com/6502/gojson"

var cfg Config
err := json.UnmarshalWith(data, &cfg, json.Relaxed)
```

```jsonc
// config.jsonc — accepted with json.Relaxed
{
    name: "server",          /* unquoted keys */
    ports: [8080, 8081,],    // trailing commas
    timeout: Infinity,       // non-finite numbers
    ratio: NaN,
}
```

## Requirements

- Go 1.25 or newer.
- No dependencies outside the standard library.

```sh
go get github.com/6502/gojson
```

## Extensions

Each extension is a flag of type `Options`. Combine them with `|`.

| Option                | Accepts                                                     | Example                      |
|-----------------------|-------------------------------------------------------------|------------------------------|
| `AllowTrailingCommas` | A comma after the last element of an array or object        | `[1, 2,]`, `{"a": 1,}`       |
| `AllowUnquotedKeys`   | Object keys that are identifiers `[a-zA-Z_$][a-zA-Z0-9_$]*` | `{name: 1, _id: 2, $ref: 3}` |
| `AllowComments`       | `// line` and `/* block */` comments wherever whitespace is allowed | `[1, /* two */ 2] // end` |
| `AllowNonFinite`      | The literals `NaN`, `Infinity`, `+Infinity`, `-Infinity`    | `{"x": -Infinity}`           |
| `Relaxed`             | All of the above                                            |                              |

The zero value (`Options(0)`) means strict, standard JSON.

What stays invalid even with every extension on:

- Empty elements: `[,]`, `[1,,2]`, `{,}`.
- Keys starting with a digit or containing other characters: `{1a: 2}`, `{a-b: 2}`.
- Unquoted strings as values: `{"a": b}`.
- Single-quoted strings, hex numbers, and other JSON5 features not listed above.
- Unterminated block comments.

## API

### Decoding

```go
// Strict: identical to encoding/json.
func Unmarshal(data []byte, v any) error
func Valid(data []byte) bool

// With extensions.
func UnmarshalWith(data []byte, v any, opts Options) error
func ValidWith(data []byte, opts Options) bool
```

### Streaming

`Decoder` has the full `encoding/json` API (`Decode`, `Token`, `More`,
`Buffered`, `InputOffset`, `UseNumber`, `DisallowUnknownFields`) plus
`SetOptions`:

```go
dec := json.NewDecoder(r)
dec.SetOptions(json.AllowComments | json.AllowTrailingCommas)
for {
    var v Item
    if err := dec.Decode(&v); err == io.EOF {
        break
    } else if err != nil {
        return err
    }
    // ...
}
```

Comments between top-level values are skipped. `Token` returns unquoted keys
as strings and accepts trailing commas. A comment left open at the end of the
stream gives `io.ErrUnexpectedEOF`.

### Converting to standard JSON

```go
func Standardize(data []byte, opts Options) ([]byte, error)
```

`Standardize` checks `data` under `opts`, then removes comments and trailing
commas and quotes unquoted keys. Everything else, including whitespace, is
left as it was. The result can be passed to any standard JSON tool. Standard
JSON has no way to write `NaN` or `Infinity`, so `Standardize` returns an
error if the input contains them.

### Encoding and helpers

`Marshal`, `MarshalIndent` and `NewEncoder` call `encoding/json` directly.
`Compact`, `Indent` and `HTMLEscape` are the standard library versions, vendored
so they return this package's `SyntaxError`. They accept only standard JSON;
run relaxed input through `Standardize` first.

### Shared types

These are type aliases of the `encoding/json` types, so values and errors can
be mixed freely with code written against the standard library:

`Unmarshaler`, `Marshaler`, `Number`, `RawMessage`, `Delim`, `Token`,
`Encoder`, `UnmarshalTypeError`, `InvalidUnmarshalError`, `MarshalerError`,
`UnsupportedTypeError`, `UnsupportedValueError` (and the deprecated
`UnmarshalFieldError`, `InvalidUTF8Error`).

## Behavior details

**Compatibility.** Mapping rules, struct tags (`omitempty`, `string`, `-`),
case-insensitive field matching, embedded structs, `UnmarshalJSON` /
`UnmarshalText`, map key types, `UseNumber`, `DisallowUnknownFields`, error
values and error messages all come from the standard library code.

**`NaN` and `Infinity`.**

| Target                          | Result                                                  |
|---------------------------------|---------------------------------------------------------|
| `float32`, `float64`            | The IEEE 754 value (`math.NaN()`, `math.Inf(±1)`)       |
| `any`                           | `float64` with that value                               |
| `Number`, or `any` with `UseNumber` | The literal text, e.g. `Number("-Infinity")`; `Number.Float64()` parses it |
| Integer types                   | `*UnmarshalTypeError` (`cannot unmarshal number NaN ...`) |
| `,string` float fields          | `"NaN"`, `"Infinity"`, ... are accepted                 |

**Custom unmarshalers.** Types implementing `UnmarshalJSON`, including
`RawMessage`, receive their value rewritten to standard JSON: comments and
trailing commas are removed and unquoted keys are quoted. An `UnmarshalJSON`
method that calls the standard `json.Unmarshal` therefore keeps working.
`NaN` and `Infinity` are passed through unchanged, since standard JSON can't
express them.

**Unquoted keys and `TextUnmarshaler`.** Map keys of a type implementing
`encoding.TextUnmarshaler` receive the key text, whether it was quoted or not.

**Errors.** With no options, syntax errors have the same messages and offsets
as `encoding/json`. `SyntaxError` is this package's own type, because the
standard library's cannot be built outside it. Code that checks for
`*encoding/json.SyntaxError` must check for `*gojson.SyntaxError` instead.

## Testing

```sh
go test -vet=off .                          # unit + upstream tests
go test -vet=off -run '^$' -fuzz FuzzCompat . # differential fuzzing
```

`-vet=off` is needed because the copied upstream test files trigger vet
checks that only apply inside the standard library.

- `upstream_*_test.go`: Go's own `encoding/json` decoder, scanner, stream,
  tag and case-folding tests, run against this package.
- `relaxed_test.go`: tests for each extension, error messages, the
  `Decoder` (including reads of one byte at a time), `Token`, and
  `Standardize`.
- `compat_fuzz_test.go`: a fuzzer that checks that, on any input, results
  and errors match `encoding/json`, both with and without extensions enabled,
  and that relaxed input decodes the same as its `Standardize`d form.

## Files

| File             | Contents                                                     |
|------------------|--------------------------------------------------------------|
| `api.go`         | Package docs, `Options`, type aliases, encoding forwarders   |
| `scanner.go`     | JSON state machine, with the extensions added (forked)       |
| `decode.go`      | Reflection decoder (forked)                                  |
| `stream.go`      | `Decoder` and `Token` (forked)                               |
| `standardize.go` | `Standardize` and the rewriting used for `UnmarshalJSON`     |
| `fields.go`      | Struct field resolution, extracted from `encode.go`          |
| `fold.go`, `tags.go`, `indent.go` | Unchanged standard library helpers          |

## License

BSD 3-Clause (see `LICENSE`). The decoder, scanner, stream and helper files
are derived from Go's `encoding/json`. They keep their original copyright
headers and are covered by the Go license in `LICENSE-GO`, which is also
BSD 3-Clause.
