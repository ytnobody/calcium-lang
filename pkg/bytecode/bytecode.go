package bytecode

import (
	"encoding/binary"
	"fmt"
)

// OpCode represents a bytecode operation
type OpCode byte

const (
	// Stack operations
	OpConstant OpCode = iota // Push constant from pool
	OpPop                    // Pop top of stack
	OpDup                    // Duplicate top of stack

	// Variables
	OpGetLocal  // Load local variable
	OpSetLocal  // Store local variable
	OpGetGlobal // Load global variable
	OpSetGlobal // Store global variable

	// Arithmetic
	OpAdd // +
	OpSub // -
	OpMul // *
	OpDiv // /
	OpMod // %
	OpPow // **
	OpNeg // Unary -

	// Comparison
	OpEqual        // ==
	OpNotEqual     // !=
	OpLessThan     // <
	OpGreaterThan  // >
	OpLessEqual    // <=
	OpGreaterEqual // >=

	// Logical
	OpNot // !
	OpAnd // && (short-circuit)
	OpOr  // || (short-circuit)

	// Control flow
	OpJump        // Unconditional jump
	OpJumpIfFalse // Jump if top of stack is false
	OpJumpIfTrue  // Jump if top of stack is true

	// Functions
	OpCall       // Call function
	OpReturn     // Return from function
	OpClosure    // Create closure/function object
	OpGetFree    // Get free variable from closure
	OpCallEffect // Call effect function (func!)

	// Arrays
	OpArray       // Create array from N elements
	OpArrayConcat // Create array with concatenation
	OpIndex       // Array index access
	OpSpread      // Spread array to stack
	OpCallSpread  // Call function with spread array as arguments

	// Tuples
	OpTuple // Create tuple from N elements

	// Hash
	OpHash // Create hash from N key-value pairs

	// Objects
	OpGetMember // Get object property

	// Result handling
	OpWrapSuccess  // Wrap value in success()
	OpWrapFailure  // Wrap value in failure()
	OpUnwrapOrJump // Unwrap success or jump to failure handler
	OpIsSuccess    // Check if value is success
	OpIsFailure    // Check if value is failure
	OpUnwrap       // Unwrap success/failure

	// Built-in functions
	OpBuiltin // Call built-in function

	// Constraints
	OpCheckConstraint // Check constraint: calls constraint, wraps in success/failure

	// Modules
	OpLoadModule // Load module by name index

	// Special
	OpNull  // Push null
	OpTrue  // Push true
	OpFalse // Push false

	// Async operations
	OpStayBegin     // stay loop start (initialize state hash)
	OpStayEnd       // stay loop execution (start event loop)
	OpStayGetState  // Get stay state
	OpContinue      // async.continue (update state and continue)
	OpLeave         // async.leave (exit from stay)
	OpSpawn         // async.spawn (create task)
	OpExpects       // async.expects (create handler)
	OpHandlerReady  // handler.ready() - activate handler
	OpHandlerReset  // handler.reset()
	OpHandlerPause  // handler.pause()
	OpHandlerResume // handler.resume()
	OpCancel        // async.cancel (cancel task/handler)
	OpAll           // async.all (wait for all tasks)
	OpTimeout       // schedule.timeout (timeout event source)
	OpInterval      // schedule.interval (interval event source)

	// Tail call optimization
	OpTailCall // Tail call (reuse current frame)

	// ADT operations
	OpConstructADT // Create ADT variant: operands [tag_name_index (2), arity (1)]
	OpMatchADT     // Match ADT variant: operands [tag_name_index (2), arity (1)]

	// Function overloading
	OpAddOverload // Pop new closure (TOS) and existing closure/overloaded (TOS-1), push overloaded closure
)

// Definition describes an opcode
type Definition struct {
	Name          string
	OperandWidths []int // Width of each operand in bytes
}

var definitions = map[OpCode]*Definition{
	OpConstant: {"OpConstant", []int{2}}, // 2-byte constant index

	OpPop: {"OpPop", []int{}},
	OpDup: {"OpDup", []int{}},

	OpGetLocal:  {"OpGetLocal", []int{1}},  // 1-byte local index
	OpSetLocal:  {"OpSetLocal", []int{1}},  // 1-byte local index
	OpGetGlobal: {"OpGetGlobal", []int{2}}, // 2-byte global index
	OpSetGlobal: {"OpSetGlobal", []int{2}}, // 2-byte global index

	OpAdd: {"OpAdd", []int{}},
	OpSub: {"OpSub", []int{}},
	OpMul: {"OpMul", []int{}},
	OpDiv: {"OpDiv", []int{}},
	OpMod: {"OpMod", []int{}},
	OpPow: {"OpPow", []int{}},
	OpNeg: {"OpNeg", []int{}},

	OpEqual:        {"OpEqual", []int{}},
	OpNotEqual:     {"OpNotEqual", []int{}},
	OpLessThan:     {"OpLessThan", []int{}},
	OpGreaterThan:  {"OpGreaterThan", []int{}},
	OpLessEqual:    {"OpLessEqual", []int{}},
	OpGreaterEqual: {"OpGreaterEqual", []int{}},

	OpNot: {"OpNot", []int{}},
	OpAnd: {"OpAnd", []int{2}}, // Jump offset if false
	OpOr:  {"OpOr", []int{2}},  // Jump offset if true

	OpJump:        {"OpJump", []int{2}},        // 2-byte jump offset
	OpJumpIfFalse: {"OpJumpIfFalse", []int{2}}, // 2-byte jump offset
	OpJumpIfTrue:  {"OpJumpIfTrue", []int{2}},  // 2-byte jump offset

	OpCall:       {"OpCall", []int{1}}, // 1-byte argument count
	OpReturn:     {"OpReturn", []int{}},
	OpClosure:    {"OpClosure", []int{2, 1}}, // 2-byte function constant index, 1-byte free variable count
	OpGetFree:    {"OpGetFree", []int{1}},    // 1-byte free variable index
	OpCallEffect: {"OpCallEffect", []int{1}}, // 1-byte argument count

	OpArray:       {"OpArray", []int{2}},       // 2-byte element count
	OpArrayConcat: {"OpArrayConcat", []int{2}}, // 2-byte element count
	OpIndex:       {"OpIndex", []int{}},
	OpSpread:      {"OpSpread", []int{}},
	OpCallSpread:  {"OpCallSpread", []int{}}, // Call with spread array

	OpTuple: {"OpTuple", []int{2}}, // 2-byte element count
	OpHash:  {"OpHash", []int{2}},  // 2-byte pair count

	OpGetMember: {"OpGetMember", []int{2}}, // 2-byte property name index

	OpWrapSuccess:  {"OpWrapSuccess", []int{}},
	OpWrapFailure:  {"OpWrapFailure", []int{}},
	OpUnwrapOrJump: {"OpUnwrapOrJump", []int{2}}, // 2-byte jump offset
	OpIsSuccess:    {"OpIsSuccess", []int{}},
	OpIsFailure:    {"OpIsFailure", []int{}},
	OpUnwrap:       {"OpUnwrap", []int{}},

	OpBuiltin: {"OpBuiltin", []int{1}}, // 1-byte builtin index

	OpCheckConstraint: {"OpCheckConstraint", []int{}}, // Check constraint on stack

	OpLoadModule: {"OpLoadModule", []int{2}}, // 2-byte module path constant index

	OpNull:  {"OpNull", []int{}},
	OpTrue:  {"OpTrue", []int{}},
	OpFalse: {"OpFalse", []int{}},

	// Async operations
	OpStayBegin:     {"OpStayBegin", []int{2}},    // 2-byte number of state pairs
	OpStayEnd:       {"OpStayEnd", []int{}},       // No operands
	OpStayGetState:  {"OpStayGetState", []int{2}}, // 2-byte state key constant index
	OpContinue:      {"OpContinue", []int{2}},     // 2-byte number of state pairs to update
	OpLeave:         {"OpLeave", []int{}},         // No operands (value on stack)
	OpSpawn:         {"OpSpawn", []int{}},         // No operands (function on stack)
	OpExpects:       {"OpExpects", []int{}},       // No operands (event_source and callback on stack)
	OpHandlerReady:  {"OpHandlerReady", []int{}},  // No operands (handler on stack)
	OpHandlerReset:  {"OpHandlerReset", []int{}},  // No operands (handler on stack)
	OpHandlerPause:  {"OpHandlerPause", []int{}},  // No operands (handler on stack)
	OpHandlerResume: {"OpHandlerResume", []int{}}, // No operands (handler on stack)
	OpCancel:        {"OpCancel", []int{}},        // No operands (task/handler on stack)
	OpAll:           {"OpAll", []int{}},           // No operands (array of tasks on stack)
	OpTimeout:       {"OpTimeout", []int{}},       // No operands (ms on stack)
	OpInterval:      {"OpInterval", []int{}},      // No operands (ms on stack)

	// Tail call optimization
	OpTailCall: {"OpTailCall", []int{1}}, // 1-byte argument count

	// ADT operations
	OpConstructADT: {"OpConstructADT", []int{2, 1}}, // 2-byte tag name constant index, 1-byte arity
	OpMatchADT:     {"OpMatchADT", []int{2, 1}},     // 2-byte tag name constant index, 1-byte arity

	// Function overloading
	OpAddOverload: {"OpAddOverload", []int{}}, // No operands: pops new closure and existing, pushes overloaded
}

// Lookup returns the definition for an opcode
func Lookup(op OpCode) (*Definition, error) {
	def, ok := definitions[op]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

// Make creates a bytecode instruction
func Make(op OpCode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 1:
			instruction[offset] = byte(o)
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		}
		offset += width
	}

	return instruction
}

// ReadOperands reads operands from bytecode
func ReadOperands(def *Definition, ins []byte) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 1:
			operands[i] = int(ins[offset])
		case 2:
			operands[i] = int(binary.BigEndian.Uint16(ins[offset:]))
		}
		offset += width
	}

	return operands, offset
}

// Instructions is a slice of bytecode
type Instructions []byte

// String returns a disassembled string
func (ins Instructions) String() string {
	var out string

	i := 0
	for i < len(ins) {
		def, err := Lookup(OpCode(ins[i]))
		if err != nil {
			out += fmt.Sprintf("ERROR: %s\n", err)
			i++
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])
		out += fmt.Sprintf("%04d %s\n", i, formatInstruction(def, operands))

		i += 1 + read
	}

	return out
}

func formatInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}

	switch operandCount {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])
	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}

// Bytecode represents compiled bytecode
type Bytecode struct {
	Instructions Instructions
	Constants    []interface{} // Can hold Value or *Function
}
