package gojson

import "fmt"

// Standardize checks that data is valid JSON under opts and rewrites it as
// standard JSON: comments and trailing commas are removed and unquoted keys
// are quoted. All other bytes, including whitespace, are preserved.
//
// Since NaN and Infinity cannot be represented in standard JSON, Standardize
// returns an error if data contains them.
func Standardize(data []byte, opts Options) ([]byte, error) {
	scan := newScanner()
	defer freeScanner(scan)
	scan.opts = opts
	if err := checkValid(data, scan); err != nil {
		return nil, err
	}
	out, nonFinite := standardizeValue(data)
	if nonFinite >= 0 {
		lit := data[nonFinite:]
		for i, c := range lit {
			if c != '+' && c != '-' && !isIdentPart(c) {
				lit = lit[:i]
				break
			}
		}
		return nil, fmt.Errorf("json: %s at offset %d cannot be represented in standard JSON", lit, nonFinite)
	}
	return out, nil
}

// standardize rewrites data, which must already be valid, as standard
// JSON, except for NaN and Infinity literals which are kept.
func standardize(data []byte) []byte {
	out, _ := standardizeValue(data)
	return out
}

// standardizeValue does the work of standardize, also returning the offset
// of the first NaN or Infinity literal in data, or -1.
// data must already have been validated.
func standardizeValue(data []byte) (out []byte, nonFinite int) {
	nonFinite = -1
	out = make([]byte, 0, len(data)+16)
	for i := 0; i < len(data); {
		c := data[i]
		switch {
		case c == '"':
			j := i + 1
			for data[j] != '"' {
				if data[j] == '\\' {
					j++
				}
				j++
			}
			j++
			out = append(out, data[i:j]...)
			i = j

		case c == '/': // comment
			i += commentLen(data[i:], true)

		case c == ',':
			if j := skipSpaceAndComments(data, i+1); j < len(data) && (data[j] == ']' || data[j] == '}') {
				i++ // trailing comma: drop it
				continue
			}
			out = append(out, c)
			i++

		case isIdentStart(c) || c == '-' || c == '+' || '0' <= c && c <= '9':
			// Keyword, number, NaN/Infinity, or unquoted key.
			j := i + 1
			for j < len(data) && (isIdentPart(data[j]) || data[j] == '.' || data[j] == '+' || data[j] == '-') {
				j++
			}
			tok := data[i:j]
			if k := skipSpaceAndComments(data, j); isIdentStart(c) && k < len(data) && data[k] == ':' {
				out = append(out, '"')
				out = append(out, tok...)
				out = append(out, '"')
			} else {
				if nonFinite < 0 && isNonFinite(string(tok)) {
					nonFinite = i
				}
				out = append(out, tok...)
			}
			i = j

		default:
			out = append(out, c)
			i++
		}
	}
	return out, nonFinite
}

func skipSpaceAndComments(data []byte, i int) int {
	for i < len(data) {
		switch c := data[i]; {
		case isSpace(c):
			i++
		case c == '/':
			n := commentLen(data[i:], true)
			if n <= 0 {
				return i
			}
			i += n
		default:
			return i
		}
	}
	return i
}
