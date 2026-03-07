package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// primitiveJSONParse parses a JSON string into a Calcium value
func primitiveJSONParse(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__json_parse: expected 1 argument (json string)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__json_parse: argument must be string"))
	}
	s := args[0].AsString()
	result, pos, err := parseJSONValue(s, 0)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__json_parse: %v", err)))
	}
	// Check for trailing content (after skipping whitespace)
	pos = skipJSONWhitespace(s, pos)
	if pos < len(s) {
		return value.Failure(value.String(fmt.Sprintf("__json_parse: unexpected content at position %d", pos)))
	}
	return value.Success(result)
}

// primitiveJSONStringify converts a Calcium value to a JSON string
func primitiveJSONStringify(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__json_stringify: expected 1 argument"))
	}
	result, err := stringifyJSON(args[0])
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__json_stringify: %v", err)))
	}
	return value.Success(value.String(result))
}

// --- JSON Parser ---

func skipJSONWhitespace(s string, pos int) int {
	for pos < len(s) {
		c := s[pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			pos++
		} else {
			break
		}
	}
	return pos
}

func parseJSONValue(s string, pos int) (value.Value, int, error) {
	pos = skipJSONWhitespace(s, pos)
	if pos >= len(s) {
		return value.Null(), pos, fmt.Errorf("unexpected end of input")
	}
	switch s[pos] {
	case '"':
		return parseJSONString(s, pos)
	case '{':
		return parseJSONObject(s, pos)
	case '[':
		return parseJSONArray(s, pos)
	case 't':
		return parseJSONTrue(s, pos)
	case 'f':
		return parseJSONFalse(s, pos)
	case 'n':
		return parseJSONNull(s, pos)
	default:
		if s[pos] == '-' || (s[pos] >= '0' && s[pos] <= '9') {
			return parseJSONNumber(s, pos)
		}
		return value.Null(), pos, fmt.Errorf("unexpected character: %c at position %d", s[pos], pos)
	}
}

func parseJSONString(s string, pos int) (value.Value, int, error) {
	if pos >= len(s) || s[pos] != '"' {
		return value.Null(), pos, fmt.Errorf("expected '\"' at position %d", pos)
	}
	pos++ // skip opening quote
	var buf strings.Builder
	for pos < len(s) {
		c := s[pos]
		if c == '"' {
			return value.String(buf.String()), pos + 1, nil
		}
		if c == '\\' {
			pos++
			if pos >= len(s) {
				return value.Null(), pos, fmt.Errorf("unexpected end of string escape")
			}
			esc := s[pos]
			switch esc {
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			case '/':
				buf.WriteByte('/')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'u':
				if pos+4 >= len(s) {
					return value.Null(), pos, fmt.Errorf("incomplete unicode escape")
				}
				hex := s[pos+1 : pos+5]
				codepoint, err := strconv.ParseUint(hex, 16, 32)
				if err != nil {
					return value.Null(), pos, fmt.Errorf("invalid unicode escape: \\u%s", hex)
				}
				r := rune(codepoint)
				// Handle surrogate pairs
				if r >= 0xD800 && r <= 0xDBFF {
					if pos+10 < len(s) && s[pos+5] == '\\' && s[pos+6] == 'u' {
						hex2 := s[pos+7 : pos+11]
						low, err := strconv.ParseUint(hex2, 16, 32)
						if err == nil && low >= 0xDC00 && low <= 0xDFFF {
							r = 0x10000 + (r-0xD800)*0x400 + (rune(low) - 0xDC00)
							pos += 6
						}
					}
				}
				var ubuf [4]byte
				n := utf8.EncodeRune(ubuf[:], r)
				buf.Write(ubuf[:n])
				pos += 4
			default:
				buf.WriteByte(esc)
			}
			pos++
			continue
		}
		buf.WriteByte(c)
		pos++
	}
	return value.Null(), pos, fmt.Errorf("unterminated string")
}

func parseJSONNumber(s string, pos int) (value.Value, int, error) {
	start := pos
	if pos < len(s) && s[pos] == '-' {
		pos++
	}
	if pos >= len(s) || s[pos] < '0' || s[pos] > '9' {
		return value.Null(), pos, fmt.Errorf("invalid number at position %d", start)
	}
	for pos < len(s) && s[pos] >= '0' && s[pos] <= '9' {
		pos++
	}
	isFloat := false
	if pos < len(s) && s[pos] == '.' {
		isFloat = true
		pos++
		for pos < len(s) && s[pos] >= '0' && s[pos] <= '9' {
			pos++
		}
	}
	if pos < len(s) && (s[pos] == 'e' || s[pos] == 'E') {
		isFloat = true
		pos++
		if pos < len(s) && (s[pos] == '+' || s[pos] == '-') {
			pos++
		}
		for pos < len(s) && s[pos] >= '0' && s[pos] <= '9' {
			pos++
		}
	}
	numStr := s[start:pos]
	if isFloat {
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return value.Null(), pos, fmt.Errorf("invalid float: %s", numStr)
		}
		return value.Float(f), pos, nil
	}
	i, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(numStr, 64)
		if ferr != nil {
			return value.Null(), pos, fmt.Errorf("invalid number: %s", numStr)
		}
		return value.Float(f), pos, nil
	}
	return value.Int(i), pos, nil
}

func parseJSONTrue(s string, pos int) (value.Value, int, error) {
	if pos+4 <= len(s) && s[pos:pos+4] == "true" {
		return value.Bool(true), pos + 4, nil
	}
	return value.Null(), pos, fmt.Errorf("expected 'true' at position %d", pos)
}

func parseJSONFalse(s string, pos int) (value.Value, int, error) {
	if pos+5 <= len(s) && s[pos:pos+5] == "false" {
		return value.Bool(false), pos + 5, nil
	}
	return value.Null(), pos, fmt.Errorf("expected 'false' at position %d", pos)
}

func parseJSONNull(s string, pos int) (value.Value, int, error) {
	if pos+4 <= len(s) && s[pos:pos+4] == "null" {
		return value.Null(), pos + 4, nil
	}
	return value.Null(), pos, fmt.Errorf("expected 'null' at position %d", pos)
}

func parseJSONArray(s string, pos int) (value.Value, int, error) {
	pos++ // skip '['
	pos = skipJSONWhitespace(s, pos)
	if pos >= len(s) {
		return value.Null(), pos, fmt.Errorf("unterminated array")
	}
	if s[pos] == ']' {
		return value.Array(nil), pos + 1, nil
	}
	var elements []value.Value
	for {
		elem, next, err := parseJSONValue(s, pos)
		if err != nil {
			return value.Null(), next, err
		}
		elements = append(elements, elem)
		pos = skipJSONWhitespace(s, next)
		if pos >= len(s) {
			return value.Null(), pos, fmt.Errorf("unterminated array")
		}
		if s[pos] == ']' {
			return value.Array(elements), pos + 1, nil
		}
		if s[pos] != ',' {
			return value.Null(), pos, fmt.Errorf("expected ',' or ']' in array at position %d", pos)
		}
		pos++
		pos = skipJSONWhitespace(s, pos)
		if pos < len(s) && s[pos] == ']' {
			return value.Null(), pos, fmt.Errorf("trailing comma in array at position %d", pos)
		}
	}
}

func parseJSONObject(s string, pos int) (value.Value, int, error) {
	pos++ // skip '{'
	pos = skipJSONWhitespace(s, pos)
	if pos >= len(s) {
		return value.Null(), pos, fmt.Errorf("unterminated object")
	}
	if s[pos] == '}' {
		h := value.NewHash()
		return value.HashVal(h), pos + 1, nil
	}
	h := value.NewHash()
	for {
		if pos >= len(s) || s[pos] != '"' {
			return value.Null(), pos, fmt.Errorf("expected string key in object at position %d", pos)
		}
		keyVal, next, err := parseJSONString(s, pos)
		if err != nil {
			return value.Null(), next, err
		}
		pos = skipJSONWhitespace(s, next)
		if pos >= len(s) || s[pos] != ':' {
			return value.Null(), pos, fmt.Errorf("expected ':' after object key at position %d", pos)
		}
		pos++
		val, next2, err := parseJSONValue(s, pos)
		if err != nil {
			return value.Null(), next2, err
		}
		h.Set(keyVal, val)
		pos = skipJSONWhitespace(s, next2)
		if pos >= len(s) {
			return value.Null(), pos, fmt.Errorf("unterminated object")
		}
		if s[pos] == '}' {
			return value.HashVal(h), pos + 1, nil
		}
		if s[pos] != ',' {
			return value.Null(), pos, fmt.Errorf("expected ',' or '}' in object at position %d", pos)
		}
		pos++
		pos = skipJSONWhitespace(s, pos)
		if pos < len(s) && s[pos] == '}' {
			return value.Null(), pos, fmt.Errorf("trailing comma in object at position %d", pos)
		}
	}
}

// --- JSON Stringify ---

func stringifyJSON(v value.Value) (string, error) {
	var buf strings.Builder
	if err := writeJSON(&buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeJSON(buf *strings.Builder, v value.Value) error {
	switch v.Type {
	case value.TYPE_NULL:
		buf.WriteString("null")
	case value.TYPE_BOOL:
		if v.AsBool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case value.TYPE_INT:
		buf.WriteString(strconv.FormatInt(v.AsInt(), 10))
	case value.TYPE_FLOAT:
		f := v.AsFloat()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			buf.WriteString("null")
		} else {
			buf.WriteString(strconv.FormatFloat(f, 'f', -1, 64))
		}
	case value.TYPE_STRING:
		writeJSONString(buf, v.AsString())
	case value.TYPE_ARRAY:
		arr := v.AsArray()
		buf.WriteByte('[')
		for i, elem := range arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSON(buf, elem); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case value.TYPE_HASH:
		h := v.AsHash()
		buf.WriteByte('{')
		first := true
		for _, pair := range h.Pairs {
			if first {
				first = false
			} else {
				buf.WriteByte(',')
			}
			writeJSONString(buf, pair.Key.AsString())
			buf.WriteByte(':')
			if err := writeJSON(buf, pair.Value); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		writeJSONString(buf, v.String())
	}
	return nil
}

func writeJSONString(buf *strings.Builder, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}
