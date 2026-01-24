package builtin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// Builtin represents a built-in function
type Builtin struct {
	Name string
	Fn   value.BuiltinFn
}

// GetBuiltins returns all built-in functions
func GetBuiltins() []*Builtin {
	return []*Builtin{
		{Name: "len", Fn: builtinLen},
		{Name: "concat", Fn: builtinConcat},
		{Name: "to_string", Fn: builtinToString},
		{Name: "get", Fn: builtinGet},
		{Name: "has", Fn: builtinHas},
		{Name: "head", Fn: builtinHead},
		{Name: "tail", Fn: builtinTail},
		{Name: "push", Fn: builtinPush},
		{Name: "range", Fn: builtinRange},
	}
}

// GetBuiltinNames returns the names of all built-in functions
func GetBuiltinNames() []string {
	builtins := GetBuiltins()
	names := make([]string, len(builtins))
	for i, b := range builtins {
		names[i] = b.Name
	}
	return names
}

// len(value) - returns length of string or array
func builtinLen(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("len: expected 1 argument"))
	}

	switch args[0].Type {
	case value.TYPE_STRING:
		return value.Int(int64(len(args[0].AsString())))
	case value.TYPE_ARRAY:
		return value.Int(int64(len(args[0].AsArray())))
	default:
		return value.Failure(value.String(fmt.Sprintf("len: unsupported type %s", args[0].Type)))
	}
}

// concat(values...) - concatenates strings or arrays
func builtinConcat(args ...value.Value) value.Value {
	if len(args) == 0 {
		return value.String("")
	}

	// Check first argument type to determine concat mode
	switch args[0].Type {
	case value.TYPE_STRING:
		var sb strings.Builder
		for _, arg := range args {
			if arg.Type != value.TYPE_STRING {
				sb.WriteString(arg.String())
			} else {
				sb.WriteString(arg.AsString())
			}
		}
		return value.String(sb.String())

	case value.TYPE_ARRAY:
		var result []value.Value
		for _, arg := range args {
			if arg.Type == value.TYPE_ARRAY {
				result = append(result, arg.AsArray()...)
			} else {
				result = append(result, arg)
			}
		}
		return value.Array(result)

	default:
		// Convert all to string and concatenate
		var sb strings.Builder
		for _, arg := range args {
			sb.WriteString(arg.String())
		}
		return value.String(sb.String())
	}
}

// to_string(value) - converts value to string
func builtinToString(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("to_string: expected 1 argument"))
	}

	return value.String(args[0].String())
}

// get(array, index) or get(array, index, default) - gets element at index
func builtinGet(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 3 {
		return value.Failure(value.String("get: expected 2 or 3 arguments"))
	}

	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("get: first argument must be array"))
	}

	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("get: index must be integer"))
	}

	arr := args[0].AsArray()
	idx := int(args[1].AsInt())

	if idx < 0 || idx >= len(arr) {
		if len(args) == 3 {
			return args[2] // Return default value
		}
		return value.Null()
	}

	return arr[idx]
}

// has(array, value) - checks if array contains value
func builtinHas(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("has: expected 2 arguments"))
	}

	if args[0].Type != value.TYPE_ARRAY {
		return value.Bool(false)
	}

	arr := args[0].AsArray()
	for _, v := range arr {
		if v.Equals(args[1]) {
			return value.Bool(true)
		}
	}

	return value.Bool(false)
}

// head(array) - returns first element
func builtinHead(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("head: expected 1 argument"))
	}

	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("head: argument must be array"))
	}

	arr := args[0].AsArray()
	if len(arr) == 0 {
		return value.Null()
	}

	return arr[0]
}

// tail(array) - returns array without first element
func builtinTail(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("tail: expected 1 argument"))
	}

	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("tail: argument must be array"))
	}

	arr := args[0].AsArray()
	if len(arr) == 0 {
		return value.Array([]value.Value{})
	}

	return value.Array(arr[1:])
}

// push(array, value) - returns new array with value appended
func builtinPush(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("push: expected 2 arguments"))
	}

	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("push: first argument must be array"))
	}

	arr := args[0].AsArray()
	newArr := make([]value.Value, len(arr)+1)
	copy(newArr, arr)
	newArr[len(arr)] = args[1]

	return value.Array(newArr)
}

// range(end) or range(start, end) or range(start, end, step) - creates array of integers
func builtinRange(args ...value.Value) value.Value {
	var start, end, step int64 = 0, 0, 1

	switch len(args) {
	case 1:
		if args[0].Type != value.TYPE_INT {
			return value.Failure(value.String("range: arguments must be integers"))
		}
		end = args[0].AsInt()
	case 2:
		if args[0].Type != value.TYPE_INT || args[1].Type != value.TYPE_INT {
			return value.Failure(value.String("range: arguments must be integers"))
		}
		start = args[0].AsInt()
		end = args[1].AsInt()
	case 3:
		if args[0].Type != value.TYPE_INT || args[1].Type != value.TYPE_INT || args[2].Type != value.TYPE_INT {
			return value.Failure(value.String("range: arguments must be integers"))
		}
		start = args[0].AsInt()
		end = args[1].AsInt()
		step = args[2].AsInt()
		if step == 0 {
			return value.Failure(value.String("range: step cannot be zero"))
		}
	default:
		return value.Failure(value.String("range: expected 1, 2, or 3 arguments"))
	}

	var result []value.Value
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, value.Int(i))
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, value.Int(i))
		}
	}

	return value.Array(result)
}

// parseInt(string) - parses string to integer
func BuiltinParseInt(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("parseInt: expected 1 argument"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("parseInt: argument must be string"))
	}

	i, err := strconv.ParseInt(args[0].AsString(), 10, 64)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("parseInt: %s", err)))
	}

	return value.Int(i)
}

// parseFloat(string) - parses string to float
func BuiltinParseFloat(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("parseFloat: expected 1 argument"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("parseFloat: argument must be string"))
	}

	f, err := strconv.ParseFloat(args[0].AsString(), 64)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("parseFloat: %s", err)))
	}

	return value.Float(f)
}
