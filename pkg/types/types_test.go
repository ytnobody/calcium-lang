package types_test

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/types"
)

func TestFromName(t *testing.T) {
	cases := []struct {
		name string
		want bool // whether FromName returns non-nil
	}{
		{"Int", true},
		{"Float", true},
		{"String", true},
		{"Bool", true},
		{"Null", true},
		{"Any", true},
		{"Array", true},
		{"Hash", true},
		{"Tuple", true},
		{"Func", true},
		{"Regex", true},
		{"Success", true},
		{"Failure", true},
		{"Unknown", false},
		{"int", false},  // case-sensitive
		{"string", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := types.FromName(tc.name)
			if (got != nil) != tc.want {
				t.Errorf("FromName(%q) = %v, want non-nil=%v", tc.name, got, tc.want)
			}
		})
	}
}

func TestIsBuiltinTypeName(t *testing.T) {
	if !types.IsBuiltinTypeName("Int") {
		t.Error("Int should be a built-in type name")
	}
	if types.IsBuiltinTypeName("Positive") {
		t.Error("Positive should not be a built-in type name")
	}
}

func TestCompatibility_Any(t *testing.T) {
	// Any is compatible with everything
	others := []types.TypeExpr{
		types.Int,
		types.Float,
		types.Str,
		types.Bool,
		types.Null,
		types.Arr,
		types.Hsh,
		types.Tup,
		types.Fn,
		types.Rx,
	}
	for _, other := range others {
		if !types.Any.IsCompatibleWith(other) {
			t.Errorf("Any should be compatible with %s", other.String())
		}
	}
}

func TestCompatibility_Primitives(t *testing.T) {
	cases := []struct {
		target types.TypeExpr
		value  types.TypeExpr
		want   bool
	}{
		{types.Int, types.Int, true},
		{types.Int, types.Float, false},
		{types.Int, types.Any, true},
		{types.Float, types.Float, true},
		{types.Float, types.Int, false},
		{types.Float, types.Any, true},
		{types.Str, types.Str, true},
		{types.Str, types.Int, false},
		{types.Str, types.Any, true},
		{types.Bool, types.Bool, true},
		{types.Bool, types.Int, false},
		{types.Bool, types.Any, true},
		{types.Null, types.Null, true},
		{types.Null, types.Int, false},
		{types.Null, types.Any, true},
	}

	for _, tc := range cases {
		got := tc.target.IsCompatibleWith(tc.value)
		if got != tc.want {
			t.Errorf("%s.IsCompatibleWith(%s) = %v, want %v",
				tc.target.String(), tc.value.String(), got, tc.want)
		}
	}
}

func TestCompatibility_Array(t *testing.T) {
	arrAny := &types.ArrayType{}
	arrInt := &types.ArrayType{Elem: types.Int}
	arrFloat := &types.ArrayType{Elem: types.Float}

	if !arrAny.IsCompatibleWith(arrInt) {
		t.Error("Array<Any> should be compatible with Array<Int>")
	}
	if !arrInt.IsCompatibleWith(arrAny) {
		t.Error("Array<Int> should be compatible with Array<Any> (gradual)")
	}
	if !arrInt.IsCompatibleWith(arrInt) {
		t.Error("Array<Int> should be compatible with Array<Int>")
	}
	if arrInt.IsCompatibleWith(arrFloat) {
		t.Error("Array<Int> should NOT be compatible with Array<Float>")
	}
	if !arrAny.IsCompatibleWith(types.Any) {
		t.Error("Array should be compatible with Any")
	}
}

func TestCompatibility_Hash(t *testing.T) {
	h1 := &types.HashType{Key: types.Str, Value: types.Int}
	h2 := &types.HashType{Key: types.Str, Value: types.Float}
	h3 := &types.HashType{}

	if !h1.IsCompatibleWith(h1) {
		t.Error("same hash type should be compatible")
	}
	if h1.IsCompatibleWith(h2) {
		t.Error("Hash<String,Int> should not be compatible with Hash<String,Float>")
	}
	if !h3.IsCompatibleWith(h1) {
		t.Error("Hash<Any,Any> should be compatible with Hash<String,Int>")
	}
}

func TestCompatibility_Tuple(t *testing.T) {
	t1 := &types.TupleType{Elems: []types.TypeExpr{types.Int, types.Str}}
	t2 := &types.TupleType{Elems: []types.TypeExpr{types.Int, types.Str}}
	t3 := &types.TupleType{Elems: []types.TypeExpr{types.Int, types.Float}}
	t4 := &types.TupleType{}

	if !t1.IsCompatibleWith(t2) {
		t.Error("same tuple type should be compatible")
	}
	if t1.IsCompatibleWith(t3) {
		t.Error("(Int, String) should not be compatible with (Int, Float)")
	}
	if !t4.IsCompatibleWith(t1) {
		t.Error("empty tuple (Any) should be compatible with any tuple")
	}
}

func TestTypeStrings(t *testing.T) {
	cases := []struct {
		t    types.TypeExpr
		want string
	}{
		{types.Any, "Any"},
		{types.Int, "Int"},
		{types.Float, "Float"},
		{types.Str, "String"},
		{types.Bool, "Bool"},
		{types.Null, "Null"},
		{types.Arr, "Array"},
		{types.Hsh, "Hash"},
		{types.Tup, "Tuple"},
		{types.Fn, "Func"},
		{types.Rx, "Regex"},
		{&types.ArrayType{Elem: types.Int}, "Array<Int>"},
		{&types.HashType{Key: types.Str, Value: types.Int}, "Hash<String, Int>"},
		{&types.TupleType{Elems: []types.TypeExpr{types.Int, types.Bool}}, "Tuple<Int, Bool>"},
		{&types.SuccessType{Inner: types.Str}, "Success<String>"},
		{&types.FailureType{Inner: types.Str}, "Failure<String>"},
		{&types.NamedType{Name: "MyType"}, "MyType"},
	}

	for _, tc := range cases {
		got := tc.t.String()
		if got != tc.want {
			t.Errorf("%T.String() = %q, want %q", tc.t, got, tc.want)
		}
	}
}
