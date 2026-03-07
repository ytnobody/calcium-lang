// Package types defines the type expression system for calcium-lang's gradual typing.
//
// Gradual typing allows type annotations to be optional. Unannotated code
// is treated as having the dynamic type (Any), and type checking only applies
// where explicit annotations are present.
package types

import "strings"

// Kind represents the kind of a type expression.
type Kind int

const (
	KindAny     Kind = iota // dynamic / unknown type (the "?" of gradual typing)
	KindNull                // null
	KindBool                // bool
	KindInt                 // int
	KindFloat               // float
	KindString              // string
	KindArray               // array (homogeneous, elem type optional)
	KindHash                // hash (key/value types optional)
	KindTuple               // tuple (fixed length, element types optional)
	KindFunc                // function type
	KindRegex               // regex
	KindSuccess             // success(T)
	KindFailure             // failure(T)
	KindResult              // success(T) | failure(E)
	KindUnion               // T | U
	KindNamed               // user-defined / ADT type
)

// TypeExpr is the interface for all type expressions.
type TypeExpr interface {
	typeExpr()
	// String returns a human-readable representation of the type.
	String() string
	// IsCompatibleWith reports whether a value of type other can be
	// assigned to a binding of this type under gradual typing rules.
	// Under gradual typing, Any is compatible with everything.
	IsCompatibleWith(other TypeExpr) bool
}

// ---- primitive types ----

// AnyType is the dynamic type; compatible with everything.
type AnyType struct{}

func (*AnyType) typeExpr()    {}
func (*AnyType) String() string { return "Any" }
func (*AnyType) IsCompatibleWith(other TypeExpr) bool { return true }

// NullType represents the null type.
type NullType struct{}

func (*NullType) typeExpr()    {}
func (*NullType) String() string { return "Null" }
func (*NullType) IsCompatibleWith(other TypeExpr) bool {
	_, ok := other.(*AnyType)
	if ok {
		return true
	}
	_, ok = other.(*NullType)
	return ok
}

// BoolType represents the boolean type.
type BoolType struct{}

func (*BoolType) typeExpr()    {}
func (*BoolType) String() string { return "Bool" }
func (*BoolType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*BoolType)
	return ok
}

// IntType represents the integer type.
type IntType struct{}

func (*IntType) typeExpr()    {}
func (*IntType) String() string { return "Int" }
func (*IntType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*IntType)
	return ok
}

// FloatType represents the floating-point type.
type FloatType struct{}

func (*FloatType) typeExpr()    {}
func (*FloatType) String() string { return "Float" }
func (*FloatType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*FloatType)
	return ok
}

// StringType represents the string type.
type StringType struct{}

func (*StringType) typeExpr()    {}
func (*StringType) String() string { return "String" }
func (*StringType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*StringType)
	return ok
}

// RegexType represents the regex type.
type RegexType struct{}

func (*RegexType) typeExpr()    {}
func (*RegexType) String() string { return "Regex" }
func (*RegexType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*RegexType)
	return ok
}

// FuncType represents the function type (generic, no parameter type info).
type FuncType struct{}

func (*FuncType) typeExpr()    {}
func (*FuncType) String() string { return "Func" }
func (*FuncType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	_, ok := other.(*FuncType)
	return ok
}

// ---- composite types ----

// ArrayType represents an array of elements of a given element type.
// If Elem is nil, the element type is Any.
type ArrayType struct {
	Elem TypeExpr // nil means Array<Any>
}

func (*ArrayType) typeExpr() {}
func (a *ArrayType) String() string {
	if a.Elem == nil {
		return "Array"
	}
	return "Array<" + a.Elem.String() + ">"
}
func (a *ArrayType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*ArrayType)
	if !ok {
		return false
	}
	// If this type has no element constraint, any array is fine
	if a.Elem == nil {
		return true
	}
	// If the other has no element type, treat as Any
	if o.Elem == nil {
		return true
	}
	return a.Elem.IsCompatibleWith(o.Elem)
}

// HashType represents a hash map with given key and value types.
// If Key or Value is nil, that dimension is Any.
type HashType struct {
	Key   TypeExpr // nil means Any
	Value TypeExpr // nil means Any
}

func (*HashType) typeExpr() {}
func (h *HashType) String() string {
	if h.Key == nil && h.Value == nil {
		return "Hash"
	}
	k := "Any"
	if h.Key != nil {
		k = h.Key.String()
	}
	v := "Any"
	if h.Value != nil {
		v = h.Value.String()
	}
	return "Hash<" + k + ", " + v + ">"
}
func (h *HashType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*HashType)
	if !ok {
		return false
	}
	if h.Key != nil && o.Key != nil && !h.Key.IsCompatibleWith(o.Key) {
		return false
	}
	if h.Value != nil && o.Value != nil && !h.Value.IsCompatibleWith(o.Value) {
		return false
	}
	return true
}

// TupleType represents a fixed-length heterogeneous tuple.
// If Elems is empty, it's Tuple with any elements.
type TupleType struct {
	Elems []TypeExpr
}

func (*TupleType) typeExpr() {}
func (t *TupleType) String() string {
	if len(t.Elems) == 0 {
		return "Tuple"
	}
	parts := make([]string, len(t.Elems))
	for i, e := range t.Elems {
		parts[i] = e.String()
	}
	return "Tuple<" + strings.Join(parts, ", ") + ">"
}
func (t *TupleType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*TupleType)
	if !ok {
		return false
	}
	if len(t.Elems) == 0 || len(o.Elems) == 0 {
		return true
	}
	if len(t.Elems) != len(o.Elems) {
		return false
	}
	for i, e := range t.Elems {
		if !e.IsCompatibleWith(o.Elems[i]) {
			return false
		}
	}
	return true
}

// SuccessType represents the success(T) type.
type SuccessType struct {
	Inner TypeExpr // nil means Any
}

func (*SuccessType) typeExpr() {}
func (s *SuccessType) String() string {
	if s.Inner == nil {
		return "Success"
	}
	return "Success<" + s.Inner.String() + ">"
}
func (s *SuccessType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*SuccessType)
	if !ok {
		return false
	}
	if s.Inner == nil || o.Inner == nil {
		return true
	}
	return s.Inner.IsCompatibleWith(o.Inner)
}

// FailureType represents the failure(E) type.
type FailureType struct {
	Inner TypeExpr // nil means Any
}

func (*FailureType) typeExpr() {}
func (f *FailureType) String() string {
	if f.Inner == nil {
		return "Failure"
	}
	return "Failure<" + f.Inner.String() + ">"
}
func (f *FailureType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*FailureType)
	if !ok {
		return false
	}
	if f.Inner == nil || o.Inner == nil {
		return true
	}
	return f.Inner.IsCompatibleWith(o.Inner)
}

// NamedType represents a user-defined or ADT type by name.
type NamedType struct {
	Name string
}

func (*NamedType) typeExpr() {}
func (n *NamedType) String() string { return n.Name }
func (n *NamedType) IsCompatibleWith(other TypeExpr) bool {
	if _, ok := other.(*AnyType); ok {
		return true
	}
	o, ok := other.(*NamedType)
	if !ok {
		return false
	}
	return n.Name == o.Name
}

// ---- singleton instances for common types ----

var (
	Any    TypeExpr = &AnyType{}
	Null   TypeExpr = &NullType{}
	Bool   TypeExpr = &BoolType{}
	Int    TypeExpr = &IntType{}
	Float  TypeExpr = &FloatType{}
	Str    TypeExpr = &StringType{}
	Rx     TypeExpr = &RegexType{}
	Fn     TypeExpr = &FuncType{}
	Arr    TypeExpr = &ArrayType{}
	Hsh    TypeExpr = &HashType{}
	Tup    TypeExpr = &TupleType{}
	Succ   TypeExpr = &SuccessType{}
	Fail   TypeExpr = &FailureType{}
)

// FromName returns the TypeExpr for a given built-in type name, or nil if unknown.
// Built-in type names are: Any, Null, Bool, Int, Float, String, Array, Hash,
// Tuple, Func, Regex, Success, Failure.
func FromName(name string) TypeExpr {
	switch name {
	case "Any":
		return Any
	case "Null":
		return Null
	case "Bool":
		return Bool
	case "Int":
		return Int
	case "Float":
		return Float
	case "String":
		return Str
	case "Array":
		return Arr
	case "Hash":
		return Hsh
	case "Tuple":
		return Tup
	case "Func":
		return Fn
	case "Regex":
		return Rx
	case "Success":
		return Succ
	case "Failure":
		return Fail
	default:
		return nil
	}
}

// IsBuiltinTypeName reports whether name is a reserved built-in type name.
func IsBuiltinTypeName(name string) bool {
	return FromName(name) != nil
}
