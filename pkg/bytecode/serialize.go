package bytecode

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"regexp"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// File format constants
const (
	MagicNumber   = "CALB" // CALcium Bytecode
	VersionMajor  = 1
	VersionMinor  = 0
	FileExtension = ".bone"
)

// Constant type tags for serialization
const (
	TypeTagNull     byte = 0
	TypeTagBool     byte = 1
	TypeTagInt      byte = 2
	TypeTagFloat    byte = 3
	TypeTagString   byte = 4
	TypeTagFunction byte = 5
	TypeTagRegex    byte = 6
)

// CompiledBytecode represents a complete compiled program that can be serialized
type CompiledBytecode struct {
	Instructions Instructions
	Constants    []value.Value
}

// Serialize writes the compiled bytecode to a byte slice
func (cb *CompiledBytecode) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write magic number
	if _, err := buf.WriteString(MagicNumber); err != nil {
		return nil, err
	}

	// Write version
	if err := buf.WriteByte(VersionMajor); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(VersionMinor); err != nil {
		return nil, err
	}

	// Write number of constants
	if err := writeUint32(buf, uint32(len(cb.Constants))); err != nil {
		return nil, err
	}

	// Write each constant
	for i, c := range cb.Constants {
		if err := serializeConstant(buf, c); err != nil {
			return nil, fmt.Errorf("failed to serialize constant %d: %w", i, err)
		}
	}

	// Write instructions length
	if err := writeUint32(buf, uint32(len(cb.Instructions))); err != nil {
		return nil, err
	}

	// Write instructions
	if _, err := buf.Write(cb.Instructions); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Deserialize reads compiled bytecode from a byte slice
func Deserialize(data []byte) (*CompiledBytecode, error) {
	buf := bytes.NewReader(data)

	// Read and verify magic number
	magic := make([]byte, 4)
	if _, err := io.ReadFull(buf, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}
	if string(magic) != MagicNumber {
		return nil, fmt.Errorf("invalid file format: expected %q, got %q", MagicNumber, string(magic))
	}

	// Read and verify version
	vMajor, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read version major: %w", err)
	}
	vMinor, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read version minor: %w", err)
	}
	if vMajor > VersionMajor {
		return nil, fmt.Errorf("unsupported bytecode version: %d.%d (max supported: %d.%d)",
			vMajor, vMinor, VersionMajor, VersionMinor)
	}

	// Read number of constants
	numConstants, err := readUint32(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read constant count: %w", err)
	}

	// Read constants
	constants := make([]value.Value, numConstants)
	for i := uint32(0); i < numConstants; i++ {
		c, err := deserializeConstant(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize constant %d: %w", i, err)
		}
		constants[i] = c
	}

	// Read instructions length
	insLen, err := readUint32(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read instructions length: %w", err)
	}

	// Read instructions
	instructions := make([]byte, insLen)
	if _, err := io.ReadFull(buf, instructions); err != nil {
		return nil, fmt.Errorf("failed to read instructions: %w", err)
	}

	return &CompiledBytecode{
		Instructions: instructions,
		Constants:    constants,
	}, nil
}

func serializeConstant(w *bytes.Buffer, v value.Value) error {
	switch v.Type {
	case value.TYPE_NULL:
		return w.WriteByte(TypeTagNull)

	case value.TYPE_BOOL:
		if err := w.WriteByte(TypeTagBool); err != nil {
			return err
		}
		if v.AsBool() {
			return w.WriteByte(1)
		}
		return w.WriteByte(0)

	case value.TYPE_INT:
		if err := w.WriteByte(TypeTagInt); err != nil {
			return err
		}
		return writeInt64(w, v.AsInt())

	case value.TYPE_FLOAT:
		if err := w.WriteByte(TypeTagFloat); err != nil {
			return err
		}
		return writeFloat64(w, v.AsFloat())

	case value.TYPE_STRING:
		if err := w.WriteByte(TypeTagString); err != nil {
			return err
		}
		return writeString(w, v.AsString())

	case value.TYPE_FUNCTION:
		if err := w.WriteByte(TypeTagFunction); err != nil {
			return err
		}
		return serializeFunction(w, v.AsFunction())

	case value.TYPE_REGEX:
		if err := w.WriteByte(TypeTagRegex); err != nil {
			return err
		}
		return serializeRegex(w, v.AsRegex())

	default:
		return fmt.Errorf("unsupported constant type for serialization: %s", v.Type)
	}
}

func deserializeConstant(r *bytes.Reader) (value.Value, error) {
	tag, err := r.ReadByte()
	if err != nil {
		return value.Null(), err
	}

	switch tag {
	case TypeTagNull:
		return value.Null(), nil

	case TypeTagBool:
		b, err := r.ReadByte()
		if err != nil {
			return value.Null(), err
		}
		return value.Bool(b != 0), nil

	case TypeTagInt:
		i, err := readInt64(r)
		if err != nil {
			return value.Null(), err
		}
		return value.Int(i), nil

	case TypeTagFloat:
		f, err := readFloat64(r)
		if err != nil {
			return value.Null(), err
		}
		return value.Float(f), nil

	case TypeTagString:
		s, err := readString(r)
		if err != nil {
			return value.Null(), err
		}
		return value.String(s), nil

	case TypeTagFunction:
		fn, err := deserializeFunction(r)
		if err != nil {
			return value.Null(), err
		}
		return value.Func(fn), nil

	case TypeTagRegex:
		re, err := deserializeRegex(r)
		if err != nil {
			return value.Null(), err
		}
		return value.RegexVal(re), nil

	default:
		return value.Null(), fmt.Errorf("unknown constant type tag: %d", tag)
	}
}

func serializeFunction(w *bytes.Buffer, fn *value.Function) error {
	// Name
	if err := writeString(w, fn.Name); err != nil {
		return err
	}

	// Parameters count and names
	if err := writeUint32(w, uint32(len(fn.Parameters))); err != nil {
		return err
	}
	for _, p := range fn.Parameters {
		if err := writeString(w, p); err != nil {
			return err
		}
	}

	// NumLocals
	if err := writeUint32(w, uint32(fn.NumLocals)); err != nil {
		return err
	}

	// IsEffect
	if fn.IsEffect {
		if err := w.WriteByte(1); err != nil {
			return err
		}
	} else {
		if err := w.WriteByte(0); err != nil {
			return err
		}
	}

	// Body (bytecode)
	if err := writeUint32(w, uint32(len(fn.Body))); err != nil {
		return err
	}
	if _, err := w.Write(fn.Body); err != nil {
		return err
	}

	return nil
}

func deserializeFunction(r *bytes.Reader) (*value.Function, error) {
	fn := &value.Function{}

	// Name
	name, err := readString(r)
	if err != nil {
		return nil, err
	}
	fn.Name = name

	// Parameters
	numParams, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	fn.Parameters = make([]string, numParams)
	for i := uint32(0); i < numParams; i++ {
		p, err := readString(r)
		if err != nil {
			return nil, err
		}
		fn.Parameters[i] = p
	}

	// NumLocals
	numLocals, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	fn.NumLocals = int(numLocals)

	// IsEffect
	isEffect, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	fn.IsEffect = isEffect != 0

	// Body
	bodyLen, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	fn.Body = make([]byte, bodyLen)
	if _, err := io.ReadFull(r, fn.Body); err != nil {
		return nil, err
	}

	return fn, nil
}

func serializeRegex(w *bytes.Buffer, re *value.Regex) error {
	// Pattern
	if err := writeString(w, re.Pattern); err != nil {
		return err
	}
	// Flags
	if err := writeString(w, re.Flags); err != nil {
		return err
	}
	return nil
}

func deserializeRegex(r *bytes.Reader) (*value.Regex, error) {
	// Pattern
	pattern, err := readString(r)
	if err != nil {
		return nil, err
	}
	// Flags
	flags, err := readString(r)
	if err != nil {
		return nil, err
	}

	// Reconstruct the compiled regex
	goPattern := convertFlags(pattern, flags)
	re, err := regexp.Compile(goPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex /%s/%s: %w", pattern, flags, err)
	}

	return &value.Regex{
		Pattern: pattern,
		Flags:   flags,
		Re:      re,
	}, nil
}

// convertFlags converts Calcium regex flags to Go's (?flags) prefix format
func convertFlags(pattern, flags string) string {
	if flags == "" {
		return pattern
	}

	prefix := ""
	for _, f := range flags {
		switch f {
		case 'i', 'm', 's':
			prefix += string(f)
		}
	}

	if prefix != "" {
		return "(?" + prefix + ")" + pattern
	}
	return pattern
}

// Helper functions for binary encoding

func writeUint32(w *bytes.Buffer, v uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	_, err := w.Write(buf)
	return err
}

func readUint32(r *bytes.Reader) (uint32, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf), nil
}

func writeInt64(w *bytes.Buffer, v int64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	_, err := w.Write(buf)
	return err
}

func readInt64(r *bytes.Reader) (int64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

func writeFloat64(w *bytes.Buffer, v float64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64ToFloat64Bits(v))
	_, err := w.Write(buf)
	return err
}

func readFloat64(r *bytes.Reader) (float64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return float64FromBits(binary.BigEndian.Uint64(buf)), nil
}

func writeString(w *bytes.Buffer, s string) error {
	if err := writeUint32(w, uint32(len(s))); err != nil {
		return err
	}
	_, err := w.WriteString(s)
	return err
}

func readString(r *bytes.Reader) (string, error) {
	length, err := readUint32(r)
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// float64 bit conversion using math package
func uint64ToFloat64Bits(f float64) uint64 {
	return math.Float64bits(f)
}

func float64FromBits(b uint64) float64 {
	return math.Float64frombits(b)
}
