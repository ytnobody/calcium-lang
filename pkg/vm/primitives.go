package vm

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

// Primitives are low-level functions that cannot be implemented in Calcium.
// They are prefixed with __ to indicate they are internal.
// These primitives are used by the standard library (stdlib/core/*.ca).

// =============================================================================
// I/O Primitives
// =============================================================================

// __print prints a string to stdout (no newline)
func primitiveprint(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__print: expected 1 argument"))
	}
	fmt.Print(args[0].String())
	return value.Null()
}

// __println prints a string to stdout with newline
func primitivePrintln(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__println: expected 1 argument"))
	}
	fmt.Println(args[0].String())
	return value.Null()
}

// __read_file reads a file and returns its contents
func primitiveReadFile(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__read_file: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__read_file: argument must be string"))
	}
	path := args[0].AsString()
	content, err := os.ReadFile(path)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__read_file: %v", err)))
	}
	return value.Success(value.String(string(content)))
}

// __write_file writes content to a file
func primitiveWriteFile(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__write_file: expected 2 arguments (path, content)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__write_file: path must be string"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__write_file: content must be string"))
	}
	path := args[0].AsString()
	content := args[1].AsString()
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__write_file: %v", err)))
	}
	return value.Success(value.Null())
}

// =============================================================================
// Math Primitives
// =============================================================================

// __floor returns the floor of a number
func primitiveFloor(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__floor: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__floor: argument must be number"))
	}
	return value.Int(int64(math.Floor(n)))
}

// __ceil returns the ceiling of a number
func primitiveCeil(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__ceil: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__ceil: argument must be number"))
	}
	return value.Int(int64(math.Ceil(n)))
}

// __round returns the rounded value of a number
func primitiveRound(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__round: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__round: argument must be number"))
	}
	return value.Int(int64(math.Round(n)))
}

// __sqrt returns the square root of a number
func primitiveSqrt(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__sqrt: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__sqrt: argument must be number"))
	}
	if n < 0 {
		return value.Failure(value.String("__sqrt: cannot take square root of negative number"))
	}
	return value.Float(math.Sqrt(n))
}

// __pow returns base raised to the power of exp
func primitivePow(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__pow: expected 2 arguments (base, exp)"))
	}
	base, ok1 := args[0].ToNumber()
	exp, ok2 := args[1].ToNumber()
	if !ok1 || !ok2 {
		return value.Failure(value.String("__pow: arguments must be numbers"))
	}
	return value.Float(math.Pow(base, exp))
}

// __sin returns the sine of a number (radians)
func primitiveSin(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__sin: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__sin: argument must be number"))
	}
	return value.Float(math.Sin(n))
}

// __cos returns the cosine of a number (radians)
func primitiveCos(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__cos: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__cos: argument must be number"))
	}
	return value.Float(math.Cos(n))
}

// __tan returns the tangent of a number (radians)
func primitiveTan(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__tan: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__tan: argument must be number"))
	}
	return value.Float(math.Tan(n))
}

// __log returns the natural logarithm of a number
func primitiveLog(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__log: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__log: argument must be number"))
	}
	if n <= 0 {
		return value.Failure(value.String("__log: argument must be positive"))
	}
	return value.Float(math.Log(n))
}

// __exp returns e raised to the power of a number
func primitiveExp(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__exp: expected 1 argument"))
	}
	n, ok := args[0].ToNumber()
	if !ok {
		return value.Failure(value.String("__exp: argument must be number"))
	}
	return value.Float(math.Exp(n))
}

// =============================================================================
// String Primitives
// =============================================================================

// __str_len returns the length of a string (in runes/characters)
func primitiveStrLen(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__str_len: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_len: argument must be string"))
	}
	return value.Int(int64(len([]rune(args[0].AsString()))))
}

// __str_split splits a string by a separator
func primitiveStrSplit(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_split: expected 2 arguments (string, separator)"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_split: arguments must be strings"))
	}
	str := args[0].AsString()
	sep := args[1].AsString()
	parts := strings.Split(str, sep)
	result := make([]value.Value, len(parts))
	for i, p := range parts {
		result[i] = value.String(p)
	}
	return value.Array(result)
}

// __str_join joins an array of strings with a separator
func primitiveStrJoin(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_join: expected 2 arguments (array, separator)"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("__str_join: first argument must be array"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_join: second argument must be string"))
	}
	arr := args[0].AsArray()
	sep := args[1].AsString()
	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = v.String()
	}
	return value.String(strings.Join(strs, sep))
}

// __str_trim removes leading and trailing whitespace
func primitiveStrTrim(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__str_trim: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_trim: argument must be string"))
	}
	return value.String(strings.TrimSpace(args[0].AsString()))
}

// __str_upper converts a string to uppercase
func primitiveStrUpper(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__str_upper: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_upper: argument must be string"))
	}
	return value.String(strings.ToUpper(args[0].AsString()))
}

// __str_lower converts a string to lowercase
func primitiveStrLower(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__str_lower: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_lower: argument must be string"))
	}
	return value.String(strings.ToLower(args[0].AsString()))
}

// __str_replace replaces all occurrences of old with new
func primitiveStrReplace(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("__str_replace: expected 3 arguments (string, old, new)"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING || args[2].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_replace: arguments must be strings"))
	}
	str := args[0].AsString()
	old := args[1].AsString()
	new := args[2].AsString()
	return value.String(strings.ReplaceAll(str, old, new))
}

// __str_substring extracts a substring
func primitiveStrSubstring(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 3 {
		return value.Failure(value.String("__str_substring: expected 2-3 arguments (string, start, [end])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_substring: first argument must be string"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("__str_substring: start must be integer"))
	}

	runes := []rune(args[0].AsString())
	start := int(args[1].AsInt())
	end := len(runes)

	if len(args) == 3 {
		if args[2].Type != value.TYPE_INT {
			return value.Failure(value.String("__str_substring: end must be integer"))
		}
		end = int(args[2].AsInt())
	}

	// Handle negative indices
	if start < 0 {
		start = len(runes) + start
	}
	if end < 0 {
		end = len(runes) + end
	}

	// Bounds check
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		return value.String("")
	}

	return value.String(string(runes[start:end]))
}

// __str_char_at returns the character at a given index
func primitiveStrCharAt(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_char_at: expected 2 arguments (string, index)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_char_at: first argument must be string"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("__str_char_at: index must be integer"))
	}

	runes := []rune(args[0].AsString())
	idx := int(args[1].AsInt())

	// Handle negative index
	if idx < 0 {
		idx = len(runes) + idx
	}

	if idx < 0 || idx >= len(runes) {
		return value.Failure(value.String("__str_char_at: index out of bounds"))
	}

	return value.String(string(runes[idx]))
}

// __str_index_of finds the index of a substring
func primitiveStrIndexOf(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_index_of: expected 2 arguments (string, substring)"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_index_of: arguments must be strings"))
	}
	str := args[0].AsString()
	substr := args[1].AsString()
	idx := strings.Index(str, substr)
	return value.Int(int64(idx))
}

// __str_starts_with checks if a string starts with a prefix
func primitiveStrStartsWith(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_starts_with: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_starts_with: arguments must be strings"))
	}
	return value.Bool(strings.HasPrefix(args[0].AsString(), args[1].AsString()))
}

// __str_ends_with checks if a string ends with a suffix
func primitiveStrEndsWith(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_ends_with: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_ends_with: arguments must be strings"))
	}
	return value.Bool(strings.HasSuffix(args[0].AsString(), args[1].AsString()))
}

// __str_contains checks if a string contains a substring
func primitiveStrContains(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__str_contains: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__str_contains: arguments must be strings"))
	}
	return value.Bool(strings.Contains(args[0].AsString(), args[1].AsString()))
}

// =============================================================================
// Regex Primitives
// =============================================================================

// __regex_test checks if a string matches a regex
func primitiveRegexTest(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__regex_test: expected 2 arguments (string, regex)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_test: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_test: second argument must be regex"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	return value.Bool(re.Re.MatchString(str))
}

// __regex_find finds the first match
func primitiveRegexFind(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__regex_find: expected 2 arguments (string, regex)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_find: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_find: second argument must be regex"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	match := re.Re.FindString(str)
	if match == "" {
		return value.Failure(value.String("no match"))
	}
	return value.Success(value.String(match))
}

// __regex_find_all finds all matches
func primitiveRegexFindAll(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__regex_find_all: expected 2 arguments (string, regex)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_find_all: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_find_all: second argument must be regex"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	matches := re.Re.FindAllString(str, -1)
	result := make([]value.Value, len(matches))
	for i, m := range matches {
		result[i] = value.String(m)
	}
	return value.Array(result)
}

// __regex_replace replaces all matches with a replacement string
func primitiveRegexReplace(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("__regex_replace: expected 3 arguments (string, regex, replacement)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_replace: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_replace: second argument must be regex"))
	}
	if args[2].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_replace: third argument must be string"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	repl := args[2].AsString()
	return value.String(re.Re.ReplaceAllString(str, repl))
}

// __regex_replace_first replaces only the first match with a replacement string
func primitiveRegexReplaceFirst(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("__regex_replace_first: expected 3 arguments (string, regex, replacement)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_replace_first: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_replace_first: second argument must be regex"))
	}
	if args[2].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_replace_first: third argument must be string"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	repl := args[2].AsString()

	// Find the first match
	loc := re.Re.FindStringIndex(str)
	if loc == nil {
		// No match, return original string
		return value.String(str)
	}

	// Replace only the first match
	result := str[:loc[0]] + repl + str[loc[1]:]
	return value.String(result)
}

// __regex_split splits a string by a regex pattern
func primitiveRegexSplit(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__regex_split: expected 2 arguments (string, regex)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_split: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_split: second argument must be regex"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	parts := re.Re.Split(str, -1)
	result := make([]value.Value, len(parts))
	for i, p := range parts {
		result[i] = value.String(p)
	}
	return value.Array(result)
}

// __regex_capture returns capture groups from a match
func primitiveRegexCapture(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__regex_capture: expected 2 arguments (string, regex)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_capture: first argument must be string"))
	}
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("__regex_capture: second argument must be regex"))
	}
	str := args[0].AsString()
	re := args[1].AsRegex()
	matches := re.Re.FindStringSubmatch(str)
	if matches == nil {
		return value.Failure(value.String("no match"))
	}
	result := make([]value.Value, len(matches))
	for i, m := range matches {
		result[i] = value.String(m)
	}
	return value.Success(value.Array(result))
}

// __regex_compile compiles a regex pattern
func primitiveRegexCompile(args ...value.Value) value.Value {
	if len(args) < 1 || len(args) > 2 {
		return value.Failure(value.String("__regex_compile: expected 1-2 arguments (pattern, [flags])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__regex_compile: pattern must be string"))
	}
	pattern := args[0].AsString()
	flags := ""
	if len(args) == 2 {
		if args[1].Type != value.TYPE_STRING {
			return value.Failure(value.String("__regex_compile: flags must be string"))
		}
		flags = args[1].AsString()
	}

	// Convert flags to Go regex prefix
	goPattern := pattern
	if flags != "" {
		prefix := ""
		for _, f := range flags {
			switch f {
			case 'i', 'm', 's':
				prefix += string(f)
			}
		}
		if prefix != "" {
			goPattern = "(?" + prefix + ")" + pattern
		}
	}

	re, err := regexp.Compile(goPattern)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__regex_compile: %v", err)))
	}

	return value.Success(value.RegexVal(&value.Regex{
		Pattern: pattern,
		Flags:   flags,
		Re:      re,
	}))
}

// =============================================================================
// Type Conversion Primitives
// =============================================================================

// __to_int converts a value to integer
// Returns success(int) on success, failure(message) on error
func primitiveToInt(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__to_int: expected 1 argument"))
	}
	switch args[0].Type {
	case value.TYPE_INT:
		return value.Success(args[0])
	case value.TYPE_FLOAT:
		return value.Success(value.Int(int64(args[0].AsFloat())))
	case value.TYPE_STRING:
		var i int64
		_, err := fmt.Sscanf(args[0].AsString(), "%d", &i)
		if err != nil {
			return value.Failure(value.String(fmt.Sprintf("__to_int: cannot parse %q as int", args[0].AsString())))
		}
		return value.Success(value.Int(i))
	case value.TYPE_BOOL:
		if args[0].AsBool() {
			return value.Success(value.Int(1))
		}
		return value.Success(value.Int(0))
	default:
		return value.Failure(value.String(fmt.Sprintf("__to_int: cannot convert %s to int", args[0].Type)))
	}
}

// __to_float converts a value to float
// Returns success(float) on success, failure(message) on error
func primitiveToFloat(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__to_float: expected 1 argument"))
	}
	switch args[0].Type {
	case value.TYPE_FLOAT:
		return value.Success(args[0])
	case value.TYPE_INT:
		return value.Success(value.Float(float64(args[0].AsInt())))
	case value.TYPE_STRING:
		var f float64
		_, err := fmt.Sscanf(args[0].AsString(), "%f", &f)
		if err != nil {
			return value.Failure(value.String(fmt.Sprintf("__to_float: cannot parse %q as float", args[0].AsString())))
		}
		return value.Success(value.Float(f))
	default:
		return value.Failure(value.String(fmt.Sprintf("__to_float: cannot convert %s to float", args[0].Type)))
	}
}

// __type_of returns the type of a value as a string
func primitiveTypeOf(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__type_of: expected 1 argument"))
	}
	return value.String(args[0].Type.String())
}

// __null returns the null value
func primitiveNull(args ...value.Value) value.Value {
	return value.Null()
}

// =============================================================================
// HTTP Primitives
// =============================================================================

// httpClient is the default HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// __http_get performs a GET request
func primitiveHttpGet(args ...value.Value) value.Value {
	if len(args) < 1 || len(args) > 2 {
		return value.Failure(value.String("__http_get: expected 1-2 arguments (url, [headers])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_get: url must be string"))
	}

	url := args[0].AsString()
	var headers map[string]string
	if len(args) == 2 && args[1].Type == value.TYPE_HASH {
		headers = hashToStringMap(args[1].AsHash())
	}

	return doHttpRequest("GET", url, headers, "")
}

// __http_post performs a POST request
func primitiveHttpPost(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 4 {
		return value.Failure(value.String("__http_post: expected 2-4 arguments (url, body, [content_type], [headers])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_post: url must be string"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_post: body must be string"))
	}

	url := args[0].AsString()
	body := args[1].AsString()
	contentType := "application/json"
	var headers map[string]string

	if len(args) >= 3 && args[2].Type == value.TYPE_STRING {
		contentType = args[2].AsString()
	}
	if len(args) >= 4 && args[3].Type == value.TYPE_HASH {
		headers = hashToStringMap(args[3].AsHash())
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = contentType

	return doHttpRequestWithBody("POST", url, headers, body)
}

// __http_put performs a PUT request
func primitiveHttpPut(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 4 {
		return value.Failure(value.String("__http_put: expected 2-4 arguments (url, body, [content_type], [headers])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_put: url must be string"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_put: body must be string"))
	}

	url := args[0].AsString()
	body := args[1].AsString()
	contentType := "application/json"
	var headers map[string]string

	if len(args) >= 3 && args[2].Type == value.TYPE_STRING {
		contentType = args[2].AsString()
	}
	if len(args) >= 4 && args[3].Type == value.TYPE_HASH {
		headers = hashToStringMap(args[3].AsHash())
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = contentType

	return doHttpRequestWithBody("PUT", url, headers, body)
}

// __http_delete performs a DELETE request
func primitiveHttpDelete(args ...value.Value) value.Value {
	if len(args) < 1 || len(args) > 2 {
		return value.Failure(value.String("__http_delete: expected 1-2 arguments (url, [headers])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_delete: url must be string"))
	}

	url := args[0].AsString()
	var headers map[string]string
	if len(args) == 2 && args[1].Type == value.TYPE_HASH {
		headers = hashToStringMap(args[1].AsHash())
	}

	return doHttpRequest("DELETE", url, headers, "")
}

// __http_request performs a generic HTTP request
func primitiveHttpRequest(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__http_request: expected 1 argument (options hash)"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String("__http_request: argument must be hash with method, url, [headers], [body]"))
	}

	opts := args[0].AsHash()

	// Get method
	methodVal, ok := opts.Get(value.String("method"))
	if !ok || methodVal.Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_request: options must include 'method' string"))
	}
	method := strings.ToUpper(methodVal.AsString())

	// Get URL
	urlVal, ok := opts.Get(value.String("url"))
	if !ok || urlVal.Type != value.TYPE_STRING {
		return value.Failure(value.String("__http_request: options must include 'url' string"))
	}
	url := urlVal.AsString()

	// Get optional headers
	var headers map[string]string
	if headersVal, ok := opts.Get(value.String("headers")); ok && headersVal.Type == value.TYPE_HASH {
		headers = hashToStringMap(headersVal.AsHash())
	}

	// Get optional body
	var body string
	if bodyVal, ok := opts.Get(value.String("body")); ok && bodyVal.Type == value.TYPE_STRING {
		body = bodyVal.AsString()
	}

	if body != "" {
		return doHttpRequestWithBody(method, url, headers, body)
	}
	return doHttpRequest(method, url, headers, "")
}

// doHttpRequest performs an HTTP request without body
func doHttpRequest(method, url string, headers map[string]string, _ string) value.Value {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("http request error: %v", err)))
	}

	// Set default User-Agent
	req.Header.Set("User-Agent", "Calcium/1.0")

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("http request error: %v", err)))
	}
	defer resp.Body.Close()

	return buildHttpResponse(resp)
}

// doHttpRequestWithBody performs an HTTP request with body
func doHttpRequestWithBody(method, url string, headers map[string]string, body string) value.Value {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("http request error: %v", err)))
	}

	// Set default User-Agent
	req.Header.Set("User-Agent", "Calcium/1.0")

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("http request error: %v", err)))
	}
	defer resp.Body.Close()

	return buildHttpResponse(resp)
}

// buildHttpResponse creates a Calcium hash from an HTTP response
func buildHttpResponse(resp *http.Response) value.Value {
	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("http read error: %v", err)))
	}

	// Build headers hash
	respHeaders := value.NewHash()
	for k, v := range resp.Header {
		if len(v) == 1 {
			respHeaders.Set(value.String(k), value.String(v[0]))
		} else {
			arr := make([]value.Value, len(v))
			for i, val := range v {
				arr[i] = value.String(val)
			}
			respHeaders.Set(value.String(k), value.Array(arr))
		}
	}

	// Build response hash
	result := value.NewHash()
	result.Set(value.String("status"), value.Int(int64(resp.StatusCode)))
	result.Set(value.String("status_text"), value.String(resp.Status))
	result.Set(value.String("headers"), value.HashVal(respHeaders))
	result.Set(value.String("body"), value.String(string(bodyBytes)))
	result.Set(value.String("ok"), value.Bool(resp.StatusCode >= 200 && resp.StatusCode < 300))

	// Return the hash directly (not wrapped in success)
	// Errors are returned as failure() values
	return value.HashVal(result)
}

// hashToStringMap converts a Calcium hash to a Go string map
func hashToStringMap(h *value.Hash) map[string]string {
	result := make(map[string]string)
	for _, pair := range h.Pairs {
		if pair.Key.Type == value.TYPE_STRING && pair.Value.Type == value.TYPE_STRING {
			result[pair.Key.AsString()] = pair.Value.AsString()
		}
	}
	return result
}

// =============================================================================
// TOML Primitives
// =============================================================================

// __toml_parse parses a TOML string into a hash
func primitiveTomlParse(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__toml_parse: expected 1 argument (toml string)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__toml_parse: argument must be string"))
	}

	tomlStr := args[0].AsString()
	var data map[string]interface{}

	if _, err := toml.Decode(tomlStr, &data); err != nil {
		return value.Failure(value.String(fmt.Sprintf("__toml_parse: %v", err)))
	}

	return value.Success(goValueToCalcium(data))
}

// __toml_stringify converts a hash to TOML string
func primitiveTomlStringify(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__toml_stringify: expected 1 argument (hash)"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String("__toml_stringify: argument must be hash"))
	}

	goVal := calciumValueToGo(args[0])
	goMap, ok := goVal.(map[string]interface{})
	if !ok {
		return value.Failure(value.String("__toml_stringify: failed to convert hash"))
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(goMap); err != nil {
		return value.Failure(value.String(fmt.Sprintf("__toml_stringify: %v", err)))
	}

	return value.Success(value.String(buf.String()))
}

// goValueToCalcium converts a Go value to a Calcium value
func goValueToCalcium(v interface{}) value.Value {
	switch val := v.(type) {
	case nil:
		return value.Null()
	case bool:
		return value.Bool(val)
	case int:
		return value.Int(int64(val))
	case int64:
		return value.Int(val)
	case float64:
		return value.Float(val)
	case string:
		return value.String(val)
	case []interface{}:
		arr := make([]value.Value, len(val))
		for i, elem := range val {
			arr[i] = goValueToCalcium(elem)
		}
		return value.Array(arr)
	case map[string]interface{}:
		hash := value.NewHash()
		for k, v := range val {
			hash.Set(value.String(k), goValueToCalcium(v))
		}
		return value.HashVal(hash)
	default:
		// Handle other types by converting to string
		return value.String(fmt.Sprintf("%v", val))
	}
}

// calciumValueToGo converts a Calcium value to a Go value
func calciumValueToGo(v value.Value) interface{} {
	switch v.Type {
	case value.TYPE_NULL:
		return nil
	case value.TYPE_BOOL:
		return v.AsBool()
	case value.TYPE_INT:
		return v.AsInt()
	case value.TYPE_FLOAT:
		return v.AsFloat()
	case value.TYPE_STRING:
		return v.AsString()
	case value.TYPE_ARRAY:
		arr := v.AsArray()
		result := make([]interface{}, len(arr))
		for i, elem := range arr {
			result[i] = calciumValueToGo(elem)
		}
		return result
	case value.TYPE_HASH:
		hash := v.AsHash()
		result := make(map[string]interface{})
		for _, pair := range hash.Pairs {
			keyStr := pair.Key.AsString()
			result[keyStr] = calciumValueToGo(pair.Value)
		}
		return result
	default:
		return nil
	}
}

// =============================================================================
// Primitive Registry
// =============================================================================

// GetPrimitives returns all primitive functions as a map
// Note: VM-specific primitives like __stdin and __eof are added separately
func GetPrimitives() map[string]*value.Builtin {
	return map[string]*value.Builtin{
		// I/O
		"__print":      {Name: "__print", Fn: primitiveprint},
		"__println":    {Name: "__println", Fn: primitivePrintln},
		"__read_file":  {Name: "__read_file", Fn: primitiveReadFile},
		"__write_file": {Name: "__write_file", Fn: primitiveWriteFile},

		// Math
		"__floor": {Name: "__floor", Fn: primitiveFloor},
		"__ceil":  {Name: "__ceil", Fn: primitiveCeil},
		"__round": {Name: "__round", Fn: primitiveRound},
		"__sqrt":  {Name: "__sqrt", Fn: primitiveSqrt},
		"__pow":   {Name: "__pow", Fn: primitivePow},
		"__sin":   {Name: "__sin", Fn: primitiveSin},
		"__cos":   {Name: "__cos", Fn: primitiveCos},
		"__tan":   {Name: "__tan", Fn: primitiveTan},
		"__log":   {Name: "__log", Fn: primitiveLog},
		"__exp":   {Name: "__exp", Fn: primitiveExp},

		// String
		"__str_len":         {Name: "__str_len", Fn: primitiveStrLen},
		"__str_split":       {Name: "__str_split", Fn: primitiveStrSplit},
		"__str_join":        {Name: "__str_join", Fn: primitiveStrJoin},
		"__str_trim":        {Name: "__str_trim", Fn: primitiveStrTrim},
		"__str_upper":       {Name: "__str_upper", Fn: primitiveStrUpper},
		"__str_lower":       {Name: "__str_lower", Fn: primitiveStrLower},
		"__str_replace":     {Name: "__str_replace", Fn: primitiveStrReplace},
		"__str_substring":   {Name: "__str_substring", Fn: primitiveStrSubstring},
		"__str_char_at":     {Name: "__str_char_at", Fn: primitiveStrCharAt},
		"__str_index_of":    {Name: "__str_index_of", Fn: primitiveStrIndexOf},
		"__str_starts_with": {Name: "__str_starts_with", Fn: primitiveStrStartsWith},
		"__str_ends_with":   {Name: "__str_ends_with", Fn: primitiveStrEndsWith},
		"__str_contains":    {Name: "__str_contains", Fn: primitiveStrContains},

		// Regex
		"__regex_test":          {Name: "__regex_test", Fn: primitiveRegexTest},
		"__regex_find":          {Name: "__regex_find", Fn: primitiveRegexFind},
		"__regex_find_all":      {Name: "__regex_find_all", Fn: primitiveRegexFindAll},
		"__regex_replace":       {Name: "__regex_replace", Fn: primitiveRegexReplace},
		"__regex_replace_first": {Name: "__regex_replace_first", Fn: primitiveRegexReplaceFirst},
		"__regex_split":         {Name: "__regex_split", Fn: primitiveRegexSplit},
		"__regex_capture":       {Name: "__regex_capture", Fn: primitiveRegexCapture},
		"__regex_compile":       {Name: "__regex_compile", Fn: primitiveRegexCompile},

		// Type conversion
		"__to_int":   {Name: "__to_int", Fn: primitiveToInt},
		"__to_float": {Name: "__to_float", Fn: primitiveToFloat},
		"__type_of":  {Name: "__type_of", Fn: primitiveTypeOf},
		"__null":     {Name: "__null", Fn: primitiveNull},

		// TOML
		"__toml_parse":     {Name: "__toml_parse", Fn: primitiveTomlParse},
		"__toml_stringify": {Name: "__toml_stringify", Fn: primitiveTomlStringify},

		// HTTP
		"__http_get":     {Name: "__http_get", Fn: primitiveHttpGet},
		"__http_post":    {Name: "__http_post", Fn: primitiveHttpPost},
		"__http_put":     {Name: "__http_put", Fn: primitiveHttpPut},
		"__http_delete":  {Name: "__http_delete", Fn: primitiveHttpDelete},
		"__http_request": {Name: "__http_request", Fn: primitiveHttpRequest},
	}
}
