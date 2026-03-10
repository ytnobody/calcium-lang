package vm

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ytnobody/calcium-lang/pkg/bytecode"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

const (
	StackSize   = 65536
	GlobalsSize = 65536
	MaxFrames   = 8192
)

// Event represents an event in the stay loop
type Event struct {
	Handler *value.Handler
	Value   value.Value
}

// StayLoop represents an active async.stay loop
type StayLoop struct {
	State      *value.Hash
	Handlers   []*value.Handler
	Tasks      []*value.Task
	EventQueue chan Event
	Done       chan value.Value
	LeaveValue value.Value
	ShouldExit bool
}

// Frame represents a call frame
type Frame struct {
	cl          *value.Closure // The closure being executed (contains fn + free vars)
	ip          int            // instruction pointer
	basePointer int            // base pointer for local variables
}

// NewFrame creates a new call frame
func NewFrame(cl *value.Closure, basePointer int) *Frame {
	return &Frame{
		cl:          cl,
		ip:          -1,
		basePointer: basePointer,
	}
}

// Instructions returns the function's bytecode
func (f *Frame) Instructions() bytecode.Instructions {
	return f.cl.Fn.Body
}

// Constants returns the function's constants if available
func (f *Frame) Constants() []value.Value {
	return f.cl.Fn.Constants
}

// HasOwnConstants returns true if the function has its own constants
func (f *Frame) HasOwnConstants() bool {
	return len(f.cl.Fn.Constants) > 0
}

// Globals returns the function's globals if available
func (f *Frame) Globals() []value.Value {
	return f.cl.Fn.Globals
}

// HasOwnGlobals returns true if the function has its own globals
func (f *Frame) HasOwnGlobals() bool {
	return len(f.cl.Fn.Globals) > 0
}

// Builtins returns the function's builtins if available
func (f *Frame) Builtins() []*value.Builtin {
	return f.cl.Fn.Builtins
}

// HasOwnBuiltins returns true if the function has its own builtins
func (f *Frame) HasOwnBuiltins() bool {
	return len(f.cl.Fn.Builtins) > 0
}

// VM is the virtual machine
type VM struct {
	constants []value.Value
	globals   []value.Value

	stack []value.Value
	sp    int // Stack pointer: always points to next free slot

	frames      []*Frame
	framesIndex int

	// Built-in functions
	builtins []*value.Builtin

	// Module registry
	modules map[string]*value.Module

	// Async support
	stayStack      []*StayLoop        // Stack of active stay loops
	taskCounter    int64              // Unique task ID counter
	handlerCounter int64              // Unique handler ID counter
	stdinSource    *value.EventSource // Shared stdin event source
	eofSource      *value.EventSource // Shared EOF event source
	stdinOnce      sync.Once          // Ensure stdin initialization happens once
	mu             sync.Mutex         // Protects shared async state

	// Source file path for external module resolution
	sourcePath string

	// Run options (import map, lock file, cached-only mode)
	runOptions *RunOptions

	// Source map for the main program (for runtime error messages)
	sourceMap *bytecode.SourceMap

	// Source input for showing source lines in error messages
	sourceInput string

	// Output writer for print/println (default: os.Stdout)
	output io.Writer
}

// SetSourcePath sets the source file path for external module resolution
func (vm *VM) SetSourcePath(path string) {
	vm.sourcePath = path
}

// SetSourceMap sets the source map for the main program
func (vm *VM) SetSourceMap(sm *bytecode.SourceMap) {
	vm.sourceMap = sm
}

// SetSourceInput sets the source code text for error messages
func (vm *VM) SetSourceInput(input string) {
	vm.sourceInput = input
}

// SetOutput sets the writer used for print/println output.
// If not set, os.Stdout is used by default.
func (vm *VM) SetOutput(w io.Writer) {
	vm.output = w
}

// getOutput returns the output writer, defaulting to os.Stdout.
func (vm *VM) getOutput() io.Writer {
	if vm.output != nil {
		return vm.output
	}
	return os.Stdout
}

// SetRunOptions sets the run options for the VM
func (vm *VM) SetRunOptions(opts *RunOptions) {
	vm.runOptions = opts
}

// GetRunOptions returns the run options for the VM
func (vm *VM) GetRunOptions() *RunOptions {
	return vm.runOptions
}

// New creates a new VM
func New(constants []value.Value) *VM {
	mainFn := &value.Function{
		Name:      "<main>",
		Body:      bytecode.Instructions{},
		NumLocals: 0,
	}
	mainClosure := &value.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	vm := &VM{
		constants:   constants,
		globals:     make([]value.Value, GlobalsSize),
		stack:       make([]value.Value, StackSize),
		sp:          0,
		frames:      frames,
		framesIndex: 1,
		modules:     make(map[string]*value.Module),
	}

	vm.initBuiltins()
	vm.initModules()

	return vm
}

// NewWithGlobals creates a VM with existing globals
func NewWithGlobals(constants []value.Value, globals []value.Value) *VM {
	vm := New(constants)
	vm.globals = globals
	return vm
}

// CloneForSpawn creates a lightweight VM clone for async.spawn execution.
// The cloned VM shares constants, globals, modules, and builtins with the parent,
// but has its own stack and frame to avoid data races during concurrent execution.
func (vm *VM) CloneForSpawn() *VM {
	mainFn := &value.Function{
		Name:      "<spawn>",
		Body:      bytecode.Instructions{},
		NumLocals: 0,
	}
	mainClosure := &value.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	clone := &VM{
		constants:   vm.constants,                   // Share constants (read-only)
		globals:     vm.globals,                     // Share globals (spawned tasks may access)
		stack:       make([]value.Value, StackSize), // Own stack
		sp:          0,
		frames:      frames, // Own frames
		framesIndex: 1,
		builtins:    vm.builtins, // Share builtins (read-only)
		modules:     vm.modules,  // Share modules (read-only)
		sourcePath:  vm.sourcePath,
		runOptions:  vm.runOptions,
	}

	return clone
}

// NewForREPL creates a VM for REPL use (doesn't capture stdin)
func NewForREPL(constants []value.Value, globals []value.Value) *VM {
	mainFn := &value.Function{
		Name:      "<main>",
		Body:      bytecode.Instructions{},
		NumLocals: 0,
	}
	mainClosure := &value.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	vm := &VM{
		constants:   constants,
		globals:     globals,
		stack:       make([]value.Value, StackSize),
		sp:          0,
		frames:      frames,
		framesIndex: 1,
		modules:     make(map[string]*value.Module),
	}

	vm.initBuiltins()
	vm.initModulesNoStdin() // Don't capture stdin in REPL mode
	return vm
}

// initModulesNoStdin initializes modules without capturing stdin
func (vm *VM) initModulesNoStdin() {
	// core.io! module - provides I/O effect functions (without stdin capture)
	ioModule := &value.Module{
		Name:    "core.io",
		Exports: make(map[string]value.Value),
	}
	ioModule.Exports["print"] = value.BuiltinFunc(&value.Builtin{Name: "print", Fn: vm.builtinPrint})
	ioModule.Exports["println"] = value.BuiltinFunc(&value.Builtin{Name: "println", Fn: vm.builtinPrintln})
	ioModule.Exports["say"] = value.BuiltinFunc(&value.Builtin{Name: "say", Fn: vm.builtinPrintln})
	ioModule.Exports["read_file"] = value.BuiltinFunc(&value.Builtin{Name: "read_file", Fn: builtinReadFile})
	ioModule.Exports["write_file"] = value.BuiltinFunc(&value.Builtin{Name: "write_file", Fn: builtinWriteFile})
	ioModule.Exports["format"] = value.BuiltinFunc(&value.Builtin{Name: "format", Fn: builtinFormat})
	ioModule.Exports["list_dir"] = value.BuiltinFunc(&value.Builtin{Name: "list_dir", Fn: builtinListDir})
	ioModule.Exports["mkdir"] = value.BuiltinFunc(&value.Builtin{Name: "mkdir", Fn: builtinMkdir})
	ioModule.Exports["exists"] = value.BuiltinFunc(&value.Builtin{Name: "exists", Fn: builtinExists})
	ioModule.Exports["delete_file"] = value.BuiltinFunc(&value.Builtin{Name: "delete_file", Fn: builtinDeleteFile})
	ioModule.Exports["file_info"] = value.BuiltinFunc(&value.Builtin{Name: "file_info", Fn: builtinFileInfo})
	// stdin/eof not available in REPL mode
	vm.modules["core.io"] = ioModule
}

func (vm *VM) initBuiltins() {
	vm.builtins = []*value.Builtin{
		{Name: "len", Fn: builtinLen},
		{Name: "concat", Fn: builtinConcat},
		{Name: "to_string", Fn: builtinToString},
		{Name: "get", Fn: builtinGet},
		{Name: "has", Fn: builtinHas},
		{Name: "head", Fn: builtinHead},
		{Name: "tail", Fn: builtinTail},
		{Name: "push", Fn: builtinPush},
		{Name: "range", Fn: builtinRange},
		// Higher-order functions (handled specially in callBuiltin)
		{Name: "map", Fn: nil},
		{Name: "filter", Fn: nil},
		{Name: "reduce", Fn: nil},
		// Hash functions
		{Name: "keys", Fn: builtinKeys},
		{Name: "values", Fn: builtinValues},
		{Name: "hash_set", Fn: builtinHashSet},
		{Name: "hash_merge", Fn: builtinHashMerge},
		// Primitive type constraint checkers
		{Name: "Int", Fn: builtinIsInt},
		{Name: "Float", Fn: builtinIsFloat},
		{Name: "String", Fn: builtinIsString},
		{Name: "Bool", Fn: builtinIsBool},
		{Name: "Array", Fn: builtinIsArray},
		{Name: "Hash", Fn: builtinIsHash},
		{Name: "Tuple", Fn: builtinIsTuple},
		{Name: "Null", Fn: builtinIsNull},
		{Name: "Function", Fn: builtinIsFunction},
		{Name: "Number", Fn: builtinIsNumber},
	}
}

func (vm *VM) initModules() {
	// core.io! module - provides I/O effect functions
	ioModule := &value.Module{
		Name:    "core.io",
		Exports: make(map[string]value.Value),
	}
	ioModule.Exports["print"] = value.BuiltinFunc(&value.Builtin{Name: "print", Fn: vm.builtinPrint})
	ioModule.Exports["println"] = value.BuiltinFunc(&value.Builtin{Name: "println", Fn: vm.builtinPrintln})
	ioModule.Exports["say"] = value.BuiltinFunc(&value.Builtin{Name: "say", Fn: vm.builtinPrintln}) // alias for println
	ioModule.Exports["read_file"] = value.BuiltinFunc(&value.Builtin{Name: "read_file", Fn: builtinReadFile})
	ioModule.Exports["write_file"] = value.BuiltinFunc(&value.Builtin{Name: "write_file", Fn: builtinWriteFile})
	ioModule.Exports["format"] = value.BuiltinFunc(&value.Builtin{Name: "format", Fn: builtinFormat})
	ioModule.Exports["list_dir"] = value.BuiltinFunc(&value.Builtin{Name: "list_dir", Fn: builtinListDir})
	ioModule.Exports["mkdir"] = value.BuiltinFunc(&value.Builtin{Name: "mkdir", Fn: builtinMkdir})
	ioModule.Exports["exists"] = value.BuiltinFunc(&value.Builtin{Name: "exists", Fn: builtinExists})
	ioModule.Exports["delete_file"] = value.BuiltinFunc(&value.Builtin{Name: "delete_file", Fn: builtinDeleteFile})
	ioModule.Exports["file_info"] = value.BuiltinFunc(&value.Builtin{Name: "file_info", Fn: builtinFileInfo})
	// Initialize stdin/eof event sources
	vm.initStdinSource()
	ioModule.Exports["stdin"] = value.EventSourceVal(vm.stdinSource)
	ioModule.Exports["eof"] = value.EventSourceVal(vm.eofSource)
	vm.modules["core.io"] = ioModule

	// Note: core.math, core.string, and core.array are loaded from Calcium stdlib
	// when first accessed via LoadStdlibModule

	// core.async! module - provides async functions
	asyncModule := &value.Module{
		Name:    "core.async",
		Exports: make(map[string]value.Value),
	}
	asyncModule.Exports["spawn"] = value.BuiltinFunc(&value.Builtin{Name: "spawn", Fn: vm.builtinAsyncSpawn})
	asyncModule.Exports["expects"] = value.BuiltinFunc(&value.Builtin{Name: "expects", Fn: vm.builtinAsyncExpects})
	asyncModule.Exports["continue"] = value.BuiltinFunc(&value.Builtin{Name: "continue", Fn: vm.builtinAsyncContinue})
	asyncModule.Exports["leave"] = value.BuiltinFunc(&value.Builtin{Name: "leave", Fn: vm.builtinAsyncLeave})
	asyncModule.Exports["cancel"] = value.BuiltinFunc(&value.Builtin{Name: "cancel", Fn: vm.builtinAsyncCancel})
	asyncModule.Exports["all"] = value.BuiltinFunc(&value.Builtin{Name: "all", Fn: vm.builtinAsyncAll})
	asyncModule.Exports["channel"] = value.BuiltinFunc(&value.Builtin{Name: "channel", Fn: vm.builtinAsyncChannel})
	vm.modules["core.async"] = asyncModule

	// core.schedule! module - provides scheduling functions
	scheduleModule := &value.Module{
		Name:    "core.schedule",
		Exports: make(map[string]value.Value),
	}
	scheduleModule.Exports["timeout"] = value.BuiltinFunc(&value.Builtin{Name: "timeout", Fn: vm.builtinScheduleTimeout})
	scheduleModule.Exports["interval"] = value.BuiltinFunc(&value.Builtin{Name: "interval", Fn: vm.builtinScheduleInterval})
	vm.modules["core.schedule"] = scheduleModule

	// Note: core.regex is loaded from Calcium stdlib when first accessed
}

// GetModule returns a module by path
func (vm *VM) GetModule(path string) (*value.Module, bool) {
	m, ok := vm.modules[path]
	return m, ok
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

// Run executes the bytecode
func (vm *VM) Run(instructions bytecode.Instructions) error {
	// Set up main frame with these instructions
	mainFn := &value.Function{
		Name:      "<main>",
		Body:      instructions,
		NumLocals: 0,
		SourceMap: vm.sourceMap,
	}
	mainClosure := &value.Closure{Fn: mainFn}
	vm.frames[0] = NewFrame(mainClosure, 0)

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip := vm.currentFrame().ip
		ins := vm.currentFrame().Instructions()
		op := bytecode.OpCode(ins[ip])

		err := vm.executeOpcode(op)
		if err != nil {
			return vm.wrapError(err)
		}
	}

	return nil
}

// wrapError enhances an error with source position and stack trace
func (vm *VM) wrapError(err error) error {
	var sb strings.Builder
	sb.WriteString(err.Error())

	// Get source position from current frame
	pos := vm.currentSourcePosition()
	if pos.IsValid() {
		sb.WriteString(fmt.Sprintf("\n  at %s", pos.String()))

		// Show the source line if available
		if vm.sourceInput != "" && pos.Line > 0 {
			line := vm.getSourceLine(pos.Line)
			if line != "" {
				sb.WriteString(fmt.Sprintf("\n  | %s", line))
				if pos.Column > 0 && pos.Column <= len(line) {
					sb.WriteString(fmt.Sprintf("\n  | %s^", strings.Repeat(" ", pos.Column-1)))
				}
			}
		}
	}

	// Build stack trace
	trace := vm.stackTrace()
	if len(trace) > 0 {
		sb.WriteString("\n\nstack trace:")
		for _, entry := range trace {
			sb.WriteString(fmt.Sprintf("\n  %s", entry))
		}
	}

	return fmt.Errorf("%s", sb.String())
}

// currentSourcePosition returns the source position for the current instruction
func (vm *VM) currentSourcePosition() bytecode.SourcePosition {
	frame := vm.currentFrame()
	sm := vm.getFrameSourceMap(frame)
	if sm == nil {
		return bytecode.SourcePosition{}
	}
	return sm.Lookup(frame.ip)
}

// getFrameSourceMap extracts the source map from a frame's function
func (vm *VM) getFrameSourceMap(frame *Frame) *bytecode.SourceMap {
	if frame == nil || frame.cl == nil || frame.cl.Fn == nil {
		return nil
	}
	if sm, ok := frame.cl.Fn.SourceMap.(*bytecode.SourceMap); ok {
		return sm
	}
	return nil
}

// stackTrace builds a stack trace from the current call frames
func (vm *VM) stackTrace() []string {
	if vm.framesIndex <= 1 {
		return nil // Only main frame, no interesting stack trace
	}

	var entries []string
	// Walk frames from current (innermost) to outermost
	for i := vm.framesIndex - 1; i >= 0; i-- {
		frame := vm.frames[i]
		if frame == nil || frame.cl == nil || frame.cl.Fn == nil {
			continue
		}

		fn := frame.cl.Fn
		funcName := fn.Name
		if funcName == "" {
			funcName = "<anonymous>"
		}

		sm := vm.getFrameSourceMap(frame)
		if sm != nil {
			pos := sm.Lookup(frame.ip)
			if pos.IsValid() {
				entries = append(entries, fmt.Sprintf("%s at %s", funcName, pos.String()))
				continue
			}
		}

		entries = append(entries, funcName)
	}

	return entries
}

// getSourceLine returns the source line at the given 1-based line number
func (vm *VM) getSourceLine(lineNum int) string {
	if vm.sourceInput == "" || lineNum <= 0 {
		return ""
	}
	lines := strings.Split(vm.sourceInput, "\n")
	if lineNum > len(lines) {
		return ""
	}
	return strings.TrimRight(lines[lineNum-1], "\r")
}

// getConstant returns the constant at the given index, using the frame's constants if available
func (vm *VM) getConstant(index uint16) value.Value {
	frame := vm.currentFrame()
	if frame.HasOwnConstants() {
		return frame.Constants()[index]
	}
	return vm.constants[index]
}

// getGlobal returns the global at the given index, using the frame's globals if available
func (vm *VM) getGlobal(index uint16) value.Value {
	frame := vm.currentFrame()
	if frame.HasOwnGlobals() {
		return frame.Globals()[index]
	}
	return vm.globals[index]
}

// setGlobal sets the global at the given index
func (vm *VM) setGlobal(index uint16, val value.Value) {
	frame := vm.currentFrame()
	if frame.HasOwnGlobals() {
		frame.Globals()[index] = val
	} else {
		vm.globals[index] = val
	}
}

// getBuiltin returns the builtin at the given index, using the frame's builtins if available
func (vm *VM) getBuiltin(index int) *value.Builtin {
	frame := vm.currentFrame()
	if frame.HasOwnBuiltins() {
		return frame.Builtins()[index]
	}
	return vm.builtins[index]
}

// executeOpcode executes a single opcode
func (vm *VM) executeOpcode(op bytecode.OpCode) error {
	ip := vm.currentFrame().ip
	ins := vm.currentFrame().Instructions()

	switch op {
	case bytecode.OpConstant:
		constIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		err := vm.push(vm.getConstant(constIndex))
		if err != nil {
			return err
		}

	case bytecode.OpPop:
		vm.pop()

	case bytecode.OpDup:
		val := vm.stack[vm.sp-1]
		err := vm.push(val)
		if err != nil {
			return err
		}

	case bytecode.OpGetLocal:
		localIndex := int(ins[ip+1])
		vm.currentFrame().ip += 1
		frame := vm.currentFrame()
		err := vm.push(vm.stack[frame.basePointer+localIndex])
		if err != nil {
			return err
		}

	case bytecode.OpSetLocal:
		localIndex := int(ins[ip+1])
		vm.currentFrame().ip += 1
		frame := vm.currentFrame()
		vm.stack[frame.basePointer+localIndex] = vm.pop()

	case bytecode.OpGetGlobal:
		globalIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		err := vm.push(vm.getGlobal(globalIndex))
		if err != nil {
			return err
		}

	case bytecode.OpSetGlobal:
		globalIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		vm.setGlobal(globalIndex, vm.pop())

	case bytecode.OpAdd, bytecode.OpSub, bytecode.OpMul, bytecode.OpDiv,
		bytecode.OpMod, bytecode.OpPow:
		err := vm.executeBinaryOp(op)
		if err != nil {
			return err
		}

	case bytecode.OpNeg:
		operand := vm.pop()
		switch operand.Type {
		case value.TYPE_INT:
			vm.push(value.Int(-operand.AsInt()))
		case value.TYPE_FLOAT:
			vm.push(value.Float(-operand.AsFloat()))
		default:
			return fmt.Errorf("type error: cannot negate value of type %s\nhint: negation '-' is only supported for int and float types", operand.Type)
		}

	case bytecode.OpEqual:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.Bool(left.Equals(right)))

	case bytecode.OpNotEqual:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.Bool(!left.Equals(right)))

	case bytecode.OpRegexMatch:
		right := vm.pop()
		left := vm.pop()
		if right.Type != value.TYPE_REGEX {
			return fmt.Errorf("type error: right operand of '~' must be a regex, got %s", right.Type)
		}
		if left.Type != value.TYPE_STRING {
			return fmt.Errorf("type error: left operand of '~' must be a string, got %s", left.Type)
		}
		re := right.AsRegex()
		vm.push(value.Bool(re.Re.MatchString(left.AsString())))

	case bytecode.OpLessThan:
		err := vm.executeComparison(op)
		if err != nil {
			return err
		}

	case bytecode.OpGreaterThan:
		err := vm.executeComparison(op)
		if err != nil {
			return err
		}

	case bytecode.OpLessEqual:
		err := vm.executeComparison(op)
		if err != nil {
			return err
		}

	case bytecode.OpGreaterEqual:
		err := vm.executeComparison(op)
		if err != nil {
			return err
		}

	case bytecode.OpNot:
		operand := vm.pop()
		vm.push(value.Bool(!operand.IsTruthy()))

	case bytecode.OpJump:
		pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip = pos - 1

	case bytecode.OpJumpIfFalse:
		pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		condition := vm.pop()
		if !condition.IsTruthy() {
			vm.currentFrame().ip = pos - 1
		}

	case bytecode.OpJumpIfTrue:
		pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		condition := vm.pop()
		if condition.IsTruthy() {
			vm.currentFrame().ip = pos - 1
		}

	case bytecode.OpCall:
		numArgs := int(ins[ip+1])
		vm.currentFrame().ip += 1
		err := vm.executeCall(numArgs)
		if err != nil {
			return err
		}

	case bytecode.OpTailCall:
		numArgs := int(ins[ip+1])
		vm.currentFrame().ip += 1
		err := vm.executeTailCall(numArgs)
		if err != nil {
			return err
		}

	case bytecode.OpConstructADT:
		tagIdx := binary.BigEndian.Uint16(ins[ip+1:])
		arity := int(ins[ip+3])
		vm.currentFrame().ip += 3

		tagStr := vm.getConstant(tagIdx).AsString()
		// tagStr is "TypeName.VariantName"
		parts := strings.SplitN(tagStr, ".", 2)
		typeName := parts[0]
		variantName := parts[1]

		vals := make([]value.Value, arity)
		for i := arity - 1; i >= 0; i-- {
			vals[i] = vm.pop()
		}

		adtVal := value.ADTVal(&value.ADT{
			TypeName: typeName,
			Tag:      variantName,
			Values:   vals,
		})
		err := vm.push(adtVal)
		if err != nil {
			return err
		}

	case bytecode.OpMatchADT:
		tagIdx := binary.BigEndian.Uint16(ins[ip+1:])
		_ = int(ins[ip+3]) // arity (for verification, not used in matching)
		vm.currentFrame().ip += 3

		expectedTag := vm.getConstant(tagIdx).AsString()

		// Pop the duplicated subject and check tag; only push bool result
		subject := vm.pop()

		if subject.Type == value.TYPE_ADT && subject.AsADT().Tag == expectedTag {
			vm.push(value.Bool(true))
		} else {
			vm.push(value.Bool(false))
		}

	case bytecode.OpCallEffect:
		numArgs := int(ins[ip+1])
		vm.currentFrame().ip += 1
		err := vm.executeCall(numArgs)
		if err != nil {
			return err
		}
		// Wrap result in success for effect functions
		result := vm.pop()
		vm.push(value.Success(result))

	case bytecode.OpReturn:
		returnValue := vm.pop()
		frame := vm.popFrame()
		vm.sp = frame.basePointer - 1

		err := vm.push(returnValue)
		if err != nil {
			return err
		}

	case bytecode.OpClosure:
		constIndex := binary.BigEndian.Uint16(ins[ip+1:])
		numFree := int(ins[ip+3])
		vm.currentFrame().ip += 3

		fn := vm.getConstant(constIndex).AsFunction()

		// If the function doesn't have its own constants, inherit from current frame
		if len(fn.Constants) == 0 && vm.currentFrame().HasOwnConstants() {
			fn.Constants = vm.currentFrame().Constants()
		} else if len(fn.Constants) == 0 {
			fn.Constants = vm.constants
		}

		// If the function doesn't have its own globals, inherit from current frame
		if len(fn.Globals) == 0 && vm.currentFrame().HasOwnGlobals() {
			fn.Globals = vm.currentFrame().Globals()
		}

		// If the function doesn't have its own builtins, inherit from current frame
		if len(fn.Builtins) == 0 && vm.currentFrame().HasOwnBuiltins() {
			fn.Builtins = vm.currentFrame().Builtins()
		}

		// Pop free variables from stack (they were pushed in order)
		free := make([]value.Value, numFree)
		for i := 0; i < numFree; i++ {
			free[i] = vm.stack[vm.sp-numFree+i]
		}
		vm.sp -= numFree

		closure := &value.Closure{Fn: fn, Free: free}
		err := vm.push(value.ClosureVal(closure))
		if err != nil {
			return err
		}

	case bytecode.OpAddOverload:
		// TOS = new closure, TOS-1 = existing value (closure or overloaded closure)
		newClosure := vm.pop().AsClosure()
		existing := vm.pop()

		var oc *value.OverloadedClosure
		switch existing.Type {
		case value.TYPE_OVERLOADED_CLOSURE:
			// Extend existing overloaded closure
			oc = existing.AsOverloadedClosure()
			oc.Variants = append(oc.Variants, newClosure)
		case value.TYPE_CLOSURE:
			// Wrap existing closure and new closure
			oc = &value.OverloadedClosure{
				Name:     newClosure.Fn.Name,
				Variants: []*value.Closure{existing.AsClosure(), newClosure},
			}
		default:
			return fmt.Errorf("internal error: OpAddOverload expected closure or overloaded_closure, got %s", existing.Type)
		}
		err := vm.push(value.OverloadedClosureVal(oc))
		if err != nil {
			return err
		}

	case bytecode.OpGetFree:
		freeIndex := int(ins[ip+1])
		vm.currentFrame().ip += 1

		currentClosure := vm.currentFrame().cl
		err := vm.push(currentClosure.Free[freeIndex])
		if err != nil {
			return err
		}

	case bytecode.OpArray:
		numElements := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		elements := make([]value.Value, numElements)
		for i := numElements - 1; i >= 0; i-- {
			elements[i] = vm.pop()
		}
		err := vm.push(value.Array(elements))
		if err != nil {
			return err
		}

	case bytecode.OpArrayConcat:
		numElements := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		var result []value.Value
		elements := make([]value.Value, numElements)
		for i := numElements - 1; i >= 0; i-- {
			elements[i] = vm.pop()
		}
		for _, elem := range elements {
			if elem.Type == value.TYPE_ARRAY {
				result = append(result, elem.AsArray()...)
			} else {
				result = append(result, elem)
			}
		}
		err := vm.push(value.Array(result))
		if err != nil {
			return err
		}

	case bytecode.OpTuple:
		numElements := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		elements := make([]value.Value, numElements)
		for i := numElements - 1; i >= 0; i-- {
			elements[i] = vm.pop()
		}
		err := vm.push(value.Tuple(elements))
		if err != nil {
			return err
		}

	case bytecode.OpHash:
		numPairs := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2

		hash := value.NewHash()
		// Pop key-value pairs from stack (in reverse order)
		pairs := make([]struct{ key, val value.Value }, numPairs)
		for i := numPairs - 1; i >= 0; i-- {
			pairs[i].val = vm.pop()
			pairs[i].key = vm.pop()
		}
		// Add pairs in original order
		for _, pair := range pairs {
			hash.Set(pair.key, pair.val)
		}
		err := vm.push(value.HashVal(hash))
		if err != nil {
			return err
		}

	case bytecode.OpIndex:
		index := vm.pop()
		left := vm.pop()

		switch left.Type {
		case value.TYPE_ARRAY:
			if index.Type != value.TYPE_INT {
				return fmt.Errorf("type error: array index must be an integer, got %s\nhint: use an integer value to access array elements (e.g., arr[0])", index.Type)
			}
			arr := left.AsArray()
			idx := int(index.AsInt())
			if idx < 0 || idx >= len(arr) {
				vm.push(value.Null())
			} else {
				vm.push(arr[idx])
			}
		case value.TYPE_TUPLE:
			if index.Type != value.TYPE_INT {
				return fmt.Errorf("type error: tuple index must be an integer, got %s\nhint: use an integer value to access tuple elements (e.g., tup[0])", index.Type)
			}
			tup := left.AsTuple()
			idx := int(index.AsInt())
			if idx < 0 || idx >= len(tup) {
				vm.push(value.Null())
			} else {
				vm.push(tup[idx])
			}
		case value.TYPE_ADT:
			if index.Type != value.TYPE_INT {
				return fmt.Errorf("type error: ADT field index must be an integer, got %s", index.Type)
			}
			adt := left.AsADT()
			idx := int(index.AsInt())
			if idx < 0 || idx >= len(adt.Values) {
				vm.push(value.Null())
			} else {
				vm.push(adt.Values[idx])
			}
		case value.TYPE_HASH:
			hash := left.AsHash()
			val, found := hash.Get(index)
			if !found {
				vm.push(value.Null())
			} else {
				vm.push(val)
			}
		default:
			return fmt.Errorf("type error: index operator '[]' is not supported for type %s\nhint: indexing is only supported for arrays, tuples, hashes, and ADT values", left.Type)
		}

	case bytecode.OpSpread:
		val := vm.pop()
		if val.Type != value.TYPE_ARRAY {
			return fmt.Errorf("type error: spread operator '...' requires an array, got %s\nhint: only arrays can be spread into function arguments or other arrays", val.Type)
		}
		arr := val.AsArray()
		for _, elem := range arr {
			err := vm.push(elem)
			if err != nil {
				return err
			}
		}

	case bytecode.OpCallSpread:
		// Stack: [func, array]
		// Pop array, spread it, call function
		arr := vm.pop()
		if arr.Type != value.TYPE_ARRAY {
			return fmt.Errorf("spread requires array, got %s", arr.Type)
		}
		elements := arr.AsArray()

		// Push array elements as arguments
		for _, elem := range elements {
			err := vm.push(elem)
			if err != nil {
				return err
			}
		}

		// Now call with number of spread elements
		err := vm.executeCall(len(elements))
		if err != nil {
			return err
		}

	case bytecode.OpGetMember:
		nameIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		propName := vm.getConstant(nameIndex).AsString()
		obj := vm.pop()

		switch obj.Type {
		case value.TYPE_MODULE:
			module := obj.AsModule()
			if val, ok := module.Exports[propName]; ok {
				vm.push(val)
			} else {
				return fmt.Errorf("module error: module '%s' has no export '%s'%s\nhint: check the module's exported names with 'use'",
					module.Name, propName, suggestSimilarName(propName, mapKeys(module.Exports)))
			}
		case value.TYPE_HASH:
			hash := obj.AsHash()
			val, found := hash.Get(value.String(propName))
			if !found {
				vm.push(value.Null())
			} else {
				vm.push(val)
			}
		case value.TYPE_TASK:
			task := obj.AsTask()
			switch propName {
			case "done":
				// Return the done event source
				vm.push(value.EventSourceVal(task.Done))
			case "result":
				vm.push(task.GetResult())
			case "status":
				statusStr := "pending"
				switch task.GetStatus() {
				case value.TaskRunning:
					statusStr = "running"
				case value.TaskCompleted:
					statusStr = "completed"
				case value.TaskCancelled:
					statusStr = "cancelled"
				case value.TaskFailed:
					statusStr = "failed"
				}
				vm.push(value.String(statusStr))
			default:
				return fmt.Errorf("property error: task has no property '%s'\nhint: available task properties are: done, result, status", propName)
			}
		case value.TYPE_HANDLER:
			handler := obj.AsHandler()
			switch propName {
			case "ready":
				// Return a builtin function that activates the handler
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "ready",
					Fn: func(args ...value.Value) value.Value {
						handler.Status = value.HandlerActive
						stay := vm.currentStay()
						if stay != nil {
							// Start listening for events from the source
							go func() {
								for {
									select {
									case val, ok := <-handler.Source.Channel:
										if !ok {
											return
										}
										if handler.Status == value.HandlerActive {
											select {
											case stay.EventQueue <- Event{Handler: handler, Value: val}:
											default:
											}
										}
									case <-handler.Source.Done:
										return
									}
								}
							}()
						}
						return value.HandlerVal(handler)
					},
				}))
			case "pause":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "pause",
					Fn: func(args ...value.Value) value.Value {
						handler.Status = value.HandlerPaused
						return value.HandlerVal(handler)
					},
				}))
			case "resume":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "resume",
					Fn: func(args ...value.Value) value.Value {
						if handler.Status == value.HandlerPaused {
							handler.Status = value.HandlerActive
						}
						return value.HandlerVal(handler)
					},
				}))
			case "reset":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "reset",
					Fn: func(args ...value.Value) value.Value {
						handler.Status = value.HandlerDormant
						return value.HandlerVal(handler)
					},
				}))
			case "status":
				statusStr := "dormant"
				switch handler.Status {
				case value.HandlerActive:
					statusStr = "active"
				case value.HandlerPaused:
					statusStr = "paused"
				case value.HandlerCancelled:
					statusStr = "cancelled"
				}
				vm.push(value.String(statusStr))
			default:
				return fmt.Errorf("property error: handler has no property '%s'\nhint: available handler properties are: ready, pause, resume, reset, status", propName)
			}
		case value.TYPE_CHANNEL:
			ch := obj.AsChannel()
			switch propName {
			case "send":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "send",
					Fn: func(args ...value.Value) value.Value {
						if len(args) != 1 {
							return value.Failure(value.String("channel.send: expected 1 argument"))
						}
						if ch.IsClosed() {
							return value.Failure(value.String("channel.send: channel is closed"))
						}
						ok := ch.Send(args[0])
						if !ok {
							return value.Failure(value.String("channel.send: channel closed during send"))
						}
						return value.Null()
					},
				}))
			case "receive":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "receive",
					Fn: func(args ...value.Value) value.Value {
						val, ok := ch.Receive()
						if !ok {
							return value.Failure(value.String("channel.receive: channel closed"))
						}
						return val
					},
				}))
			case "close":
				vm.push(value.BuiltinFunc(&value.Builtin{
					Name: "close",
					Fn: func(args ...value.Value) value.Value {
						ch.Close()
						return value.Null()
					},
				}))
			case "source":
				vm.push(value.EventSourceVal(ch.Source))
			case "status":
				if ch.IsClosed() {
					vm.push(value.String("closed"))
				} else {
					vm.push(value.String("open"))
				}
			default:
				return fmt.Errorf("property error: channel has no property '%s'\nhint: available channel properties are: send, receive, close, source, status", propName)
			}
		default:
			return fmt.Errorf("type error: cannot access property '%s' on value of type %s\nhint: member access with '.' is supported for modules, hashes, tasks, handlers, and channels", propName, obj.Type)
		}

	case bytecode.OpWrapSuccess:
		val := vm.pop()
		vm.push(value.Success(val))

	case bytecode.OpWrapFailure:
		val := vm.pop()
		vm.push(value.Failure(val))

	case bytecode.OpIsSuccess:
		val := vm.pop()
		vm.push(value.Bool(val.IsSuccess()))

	case bytecode.OpIsFailure:
		val := vm.pop()
		vm.push(value.Bool(val.IsFailure()))

	case bytecode.OpUnwrap:
		val := vm.pop()
		if val.IsSuccess() {
			vm.push(val.AsSuccess())
		} else if val.IsFailure() {
			vm.push(val.AsFailure())
		} else {
			vm.push(val)
		}

	case bytecode.OpBuiltin:
		builtinIndex := int(ins[ip+1])
		vm.currentFrame().ip += 1
		builtin := vm.getBuiltin(builtinIndex)
		vm.push(value.BuiltinFunc(builtin))

	case bytecode.OpCheckConstraint:
		// Stack after call: [saved_value, constraint_result]
		// Pop result, pop value, wrap based on result
		result := vm.pop()
		val := vm.pop()

		if result.IsTruthy() {
			vm.push(value.Success(val))
		} else {
			vm.push(value.Failure(val))
		}

	case bytecode.OpLoadModule:
		pathIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		modulePath := vm.getConstant(pathIndex).AsString()
		module, ok := vm.modules[modulePath]
		if !ok {
			// Try loading from stdlib
			stdlibMod, err := vm.LoadStdlibModule(modulePath)
			if err != nil {
				return fmt.Errorf("error loading module %s: %w", modulePath, err)
			}
			if stdlibMod != nil {
				module = stdlibMod
				vm.modules[modulePath] = module // Cache it
			} else {
				// Try loading from external filesystem
				extMod, err := vm.LoadExternalModule(modulePath)
				if err != nil {
					return fmt.Errorf("error loading module %s: %w", modulePath, err)
				}
				if extMod != nil {
					module = extMod
					vm.modules[modulePath] = module // Cache it
				} else {
					return fmt.Errorf("module error: module '%s' not found\nhint: check the module name and ensure it is installed or available in the load path", modulePath)
				}
			}
		}
		vm.push(value.ModuleVal(module))

	case bytecode.OpNull:
		vm.push(value.Null())

	case bytecode.OpTrue:
		vm.push(value.Bool(true))

	case bytecode.OpFalse:
		vm.push(value.Bool(false))

	// Async operations
	case bytecode.OpStayBegin:
		numPairs := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		err := vm.executeStayBegin(numPairs)
		if err != nil {
			return err
		}

	case bytecode.OpStayEnd:
		err := vm.executeStayEnd()
		if err != nil {
			return err
		}

	case bytecode.OpStayGetState:
		keyIndex := binary.BigEndian.Uint16(ins[ip+1:])
		vm.currentFrame().ip += 2
		key := vm.getConstant(keyIndex).AsString()
		err := vm.executeStayGetState(key)
		if err != nil {
			return err
		}

	case bytecode.OpContinue:
		numPairs := int(binary.BigEndian.Uint16(ins[ip+1:]))
		vm.currentFrame().ip += 2
		err := vm.executeContinue(numPairs)
		if err != nil {
			return err
		}

	case bytecode.OpLeave:
		err := vm.executeLeave()
		if err != nil {
			return err
		}

	case bytecode.OpSpawn:
		err := vm.executeSpawn()
		if err != nil {
			return err
		}

	case bytecode.OpExpects:
		err := vm.executeExpects()
		if err != nil {
			return err
		}

	case bytecode.OpHandlerReady:
		err := vm.executeHandlerReady()
		if err != nil {
			return err
		}

	case bytecode.OpHandlerReset:
		err := vm.executeHandlerReset()
		if err != nil {
			return err
		}

	case bytecode.OpHandlerPause:
		err := vm.executeHandlerPause()
		if err != nil {
			return err
		}

	case bytecode.OpHandlerResume:
		err := vm.executeHandlerResume()
		if err != nil {
			return err
		}

	case bytecode.OpCancel:
		err := vm.executeCancel()
		if err != nil {
			return err
		}

	case bytecode.OpAll:
		err := vm.executeAll()
		if err != nil {
			return err
		}

	case bytecode.OpTimeout:
		err := vm.executeTimeout()
		if err != nil {
			return err
		}

	case bytecode.OpInterval:
		err := vm.executeInterval()
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown opcode: %d", op)
	}
	return nil
}

func (vm *VM) push(v value.Value) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("runtime error: stack overflow\nhint: this is often caused by infinite recursion — check your recursive function for a proper base case")
	}
	vm.stack[vm.sp] = v
	vm.sp++
	return nil
}

func (vm *VM) pop() value.Value {
	v := vm.stack[vm.sp-1]
	vm.sp--
	return v
}

func (vm *VM) executeBinaryOp(op bytecode.OpCode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type
	rightType := right.Type

	// Handle string concatenation (auto-convert non-strings)
	if op == bytecode.OpAdd && (leftType == value.TYPE_STRING || rightType == value.TYPE_STRING) {
		return vm.push(value.String(left.String() + right.String()))
	}

	// Numeric operations
	if (leftType == value.TYPE_INT || leftType == value.TYPE_FLOAT) &&
		(rightType == value.TYPE_INT || rightType == value.TYPE_FLOAT) {

		// If both are ints, keep result as int (except for division)
		if leftType == value.TYPE_INT && rightType == value.TYPE_INT && op != bytecode.OpDiv {
			return vm.executeIntegerBinaryOp(op, left.AsInt(), right.AsInt())
		}

		// Otherwise, use float
		leftVal, _ := left.ToNumber()
		rightVal, _ := right.ToNumber()
		return vm.executeFloatBinaryOp(op, leftVal, rightVal)
	}

	return fmt.Errorf("type error: cannot apply binary operator to %s and %s\nhint: both operands must be numeric (int or float), or use '+' with at least one string for concatenation", leftType, rightType)
}

func (vm *VM) executeIntegerBinaryOp(op bytecode.OpCode, left, right int64) error {
	var result int64
	switch op {
	case bytecode.OpAdd:
		result = left + right
	case bytecode.OpSub:
		result = left - right
	case bytecode.OpMul:
		result = left * right
	case bytecode.OpMod:
		result = left % right
	case bytecode.OpPow:
		result = int64(math.Pow(float64(left), float64(right)))
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}
	return vm.push(value.Int(result))
}

func (vm *VM) executeFloatBinaryOp(op bytecode.OpCode, left, right float64) error {
	var result float64
	switch op {
	case bytecode.OpAdd:
		result = left + right
	case bytecode.OpSub:
		result = left - right
	case bytecode.OpMul:
		result = left * right
	case bytecode.OpDiv:
		if right == 0 {
			return fmt.Errorf("runtime error: division by zero\nhint: ensure the divisor is not zero before dividing")
		}
		result = left / right
	case bytecode.OpMod:
		result = math.Mod(left, right)
	case bytecode.OpPow:
		result = math.Pow(left, right)
	default:
		return fmt.Errorf("unknown float operator: %d", op)
	}
	return vm.push(value.Float(result))
}

func (vm *VM) executeComparison(op bytecode.OpCode) error {
	right := vm.pop()
	left := vm.pop()

	leftVal, leftOk := left.ToNumber()
	rightVal, rightOk := right.ToNumber()

	if !leftOk || !rightOk {
		return fmt.Errorf("type error: comparison operators (<, >, <=, >=) require numeric types, got %s and %s\nhint: use '==' or '!=' for equality comparison of non-numeric types", left.Type, right.Type)
	}

	var result bool
	switch op {
	case bytecode.OpLessThan:
		result = leftVal < rightVal
	case bytecode.OpGreaterThan:
		result = leftVal > rightVal
	case bytecode.OpLessEqual:
		result = leftVal <= rightVal
	case bytecode.OpGreaterEqual:
		result = leftVal >= rightVal
	}

	return vm.push(value.Bool(result))
}

func (vm *VM) executeCall(numArgs int) error {
	// The function is on the stack below the arguments
	// For pipe operator: stack is [arg, func]
	// We need to handle both cases

	callee := vm.stack[vm.sp-1-numArgs]

	switch callee.Type {
	case value.TYPE_CLOSURE:
		return vm.callClosure(callee.AsClosure(), numArgs)
	case value.TYPE_FUNCTION:
		// Wrap bare function in a closure for consistency
		cl := &value.Closure{Fn: callee.AsFunction()}
		return vm.callClosure(cl, numArgs)
	case value.TYPE_BUILTIN:
		return vm.callBuiltin(callee.AsBuiltin(), numArgs)
	case value.TYPE_PARTIAL_BUILTIN:
		return vm.callPartialBuiltin(callee.AsPartialBuiltin(), numArgs)
	case value.TYPE_OVERLOADED_CLOSURE:
		cl, err := vm.resolveOverload(callee.AsOverloadedClosure(), numArgs)
		if err != nil {
			return err
		}
		return vm.callClosure(cl, numArgs)
	default:
		return fmt.Errorf("type error: cannot call value of type %s as a function\nhint: only functions, closures, and builtins can be called with '()'", callee.Type)
	}
}

func (vm *VM) callClosure(cl *value.Closure, numArgs int) error {
	if numArgs != len(cl.Fn.Parameters) {
		return fmt.Errorf("argument error: function expects %d argument(s) but received %d\nhint: check the function definition for the correct number of parameters",
			len(cl.Fn.Parameters), numArgs)
	}

	frame := NewFrame(cl, vm.sp-numArgs)
	vm.pushFrame(frame)

	// Allocate space for locals
	vm.sp = frame.basePointer + cl.Fn.NumLocals

	return nil
}

// resolveOverload selects the appropriate variant from an OverloadedClosure
// based on argument count. If no variant matches, an error is returned.
func (vm *VM) resolveOverload(oc *value.OverloadedClosure, numArgs int) (*value.Closure, error) {
	for _, variant := range oc.Variants {
		if len(variant.Fn.Parameters) == numArgs {
			return variant, nil
		}
	}
	return nil, fmt.Errorf("argument error: no overload of function '%s' accepts %d argument(s)", oc.Name, numArgs)
}

// executeTailCall handles tail call optimization by reusing the current frame
// instead of pushing a new one, preventing stack overflow in recursive calls.
func (vm *VM) executeTailCall(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]

	switch callee.Type {
	case value.TYPE_CLOSURE:
		return vm.tailCallClosure(callee.AsClosure(), numArgs)
	case value.TYPE_FUNCTION:
		cl := &value.Closure{Fn: callee.AsFunction()}
		return vm.tailCallClosure(cl, numArgs)
	case value.TYPE_BUILTIN:
		// Builtins don't benefit from TCO; execute normally
		return vm.callBuiltin(callee.AsBuiltin(), numArgs)
	case value.TYPE_PARTIAL_BUILTIN:
		return vm.callPartialBuiltin(callee.AsPartialBuiltin(), numArgs)
	case value.TYPE_OVERLOADED_CLOSURE:
		cl, err := vm.resolveOverload(callee.AsOverloadedClosure(), numArgs)
		if err != nil {
			return err
		}
		return vm.tailCallClosure(cl, numArgs)
	default:
		return fmt.Errorf("type error: cannot call value of type %s as a function\nhint: only functions, closures, and builtins can be called with '()'", callee.Type)
	}
}

// tailCallClosure reuses the current frame for tail calls to closures.
func (vm *VM) tailCallClosure(cl *value.Closure, numArgs int) error {
	if numArgs != len(cl.Fn.Parameters) {
		return fmt.Errorf("argument error: function expects %d argument(s) but received %d\nhint: check the function definition for the correct number of parameters",
			len(cl.Fn.Parameters), numArgs)
	}

	frame := vm.currentFrame()

	// Copy arguments to the current frame's base pointer position
	// Arguments are at stack[sp-1-numArgs .. sp-1] (callee is at sp-1-numArgs)
	// We need to place them at frame.basePointer .. frame.basePointer+numArgs
	argStart := vm.sp - 1 - numArgs + 1 // skip the callee, args start after it
	for i := 0; i < numArgs; i++ {
		vm.stack[frame.basePointer+i] = vm.stack[argStart+i]
	}

	// Reset stack pointer: base pointer + number of locals
	vm.sp = frame.basePointer + cl.Fn.NumLocals

	// Update the frame to use the new closure (may be a different function in mutual recursion)
	frame.cl = cl
	frame.ip = -1 // Will be incremented to 0 at the start of the next iteration

	return nil
}

func (vm *VM) callBuiltin(builtin *value.Builtin, numArgs int) error {
	args := make([]value.Value, numArgs)
	copy(args, vm.stack[vm.sp-numArgs:vm.sp])

	// Handle higher-order functions specially
	switch builtin.Name {
	case "map":
		return vm.executeMap(args)
	case "filter":
		return vm.executeFilter(args)
	case "reduce":
		return vm.executeReduce(args)
	}

	result := builtin.Fn(args...)
	vm.sp = vm.sp - numArgs - 1 // pop args and function

	return vm.push(result)
}

func (vm *VM) callPartialBuiltin(partial *value.PartialBuiltin, numArgs int) error {
	// Combine stored args with new args
	newArgs := make([]value.Value, numArgs)
	copy(newArgs, vm.stack[vm.sp-numArgs:vm.sp])
	allArgs := append(partial.Args, newArgs...)

	// Execute the builtin with all args - we need to handle stack correctly
	// The executeXxx functions expect args on stack and adjust sp themselves
	// So we simulate having all args on stack

	var result value.Value
	var err error

	switch partial.Name {
	case "map":
		if len(allArgs) != 2 {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("map: expected 2 arguments")))
		}
		fn := allArgs[0]
		arr := allArgs[1]
		if arr.Type != value.TYPE_ARRAY {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("map: second argument must be array")))
		}
		elements := arr.AsArray()
		results := make([]value.Value, len(elements))
		for i, elem := range elements {
			r, e := vm.callValueSync(fn, []value.Value{elem})
			if e != nil {
				vm.sp = vm.sp - numArgs - 1
				return vm.push(value.Failure(value.String(fmt.Sprintf("map: %v", e))))
			}
			results[i] = r
		}
		result = value.Array(results)
	case "filter":
		if len(allArgs) != 2 {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("filter: expected 2 arguments")))
		}
		fn := allArgs[0]
		arr := allArgs[1]
		if arr.Type != value.TYPE_ARRAY {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("filter: second argument must be array")))
		}
		elements := arr.AsArray()
		var results []value.Value
		for _, elem := range elements {
			r, e := vm.callValueSync(fn, []value.Value{elem})
			if e != nil {
				vm.sp = vm.sp - numArgs - 1
				return vm.push(value.Failure(value.String(fmt.Sprintf("filter: %v", e))))
			}
			if r.IsTruthy() {
				results = append(results, elem)
			}
		}
		result = value.Array(results)
	case "reduce":
		if len(allArgs) != 3 {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("reduce: expected 3 arguments")))
		}
		fn := allArgs[0]
		accumulator := allArgs[1]
		arr := allArgs[2]
		if arr.Type != value.TYPE_ARRAY {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("reduce: third argument must be array")))
		}
		elements := arr.AsArray()
		for _, elem := range elements {
			r, e := vm.callValueSync(fn, []value.Value{accumulator, elem})
			if e != nil {
				vm.sp = vm.sp - numArgs - 1
				return vm.push(value.Failure(value.String(fmt.Sprintf("reduce: %v", e))))
			}
			accumulator = r
		}
		result = accumulator
	case "async.expects":
		// allArgs should be [callback, source] - callback was stored first, source is new arg
		if len(allArgs) != 2 {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String("async.expects: expected 2 arguments")))
		}
		callback := allArgs[0]
		source := allArgs[1]

		if source.Type != value.TYPE_EVENT_SOURCE {
			vm.sp = vm.sp - numArgs - 1
			return vm.push(value.Failure(value.String(fmt.Sprintf("async.expects: expected event_source, got %s", source.Type))))
		}

		handlerID := atomic.AddInt64(&vm.handlerCounter, 1)
		handler := &value.Handler{
			ID:       handlerID,
			Status:   value.HandlerDormant,
			Source:   source.AsEventSource(),
			Callback: callback,
		}

		// Register handler with current stay loop
		if stay := vm.currentStay(); stay != nil {
			vm.mu.Lock()
			stay.Handlers = append(stay.Handlers, handler)
			vm.mu.Unlock()
		}

		result = value.HandlerVal(handler)
	default:
		return fmt.Errorf("unknown partial builtin: %s", partial.Name)
	}

	if err != nil {
		vm.sp = vm.sp - numArgs - 1
		return vm.push(value.Failure(value.String(err.Error())))
	}

	vm.sp = vm.sp - numArgs - 1
	return vm.push(result)
}

// executeMap implements map(arr, fn) - applies fn to each element
// For pipeline: arr |> map(fn) -> map(arr, fn)
func (vm *VM) executeMap(args []value.Value) error {
	if len(args) != 2 {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("map: expected 2 arguments (array, fn)")))
	}

	arr := args[0]
	if arr.Type != value.TYPE_ARRAY {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("map: first argument must be array")))
	}

	fn := args[1]
	if fn.Type != value.TYPE_FUNCTION && fn.Type != value.TYPE_CLOSURE && fn.Type != value.TYPE_BUILTIN {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("map: second argument must be function")))
	}

	elements := arr.AsArray()
	results := make([]value.Value, len(elements))
	for i, elem := range elements {
		result, err := vm.callValueSync(fn, []value.Value{elem})
		if err != nil {
			vm.sp = vm.sp - len(args) - 1
			return vm.push(value.Failure(value.String(fmt.Sprintf("map: %v", err))))
		}
		results[i] = result
	}

	vm.sp = vm.sp - len(args) - 1
	return vm.push(value.Array(results))
}

// executeFilter implements filter(arr, pred) - keeps elements where predicate returns true
// For pipeline: arr |> filter(pred) -> filter(arr, pred)
func (vm *VM) executeFilter(args []value.Value) error {
	if len(args) != 2 {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("filter: expected 2 arguments (array, pred)")))
	}

	arr := args[0]
	if arr.Type != value.TYPE_ARRAY {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("filter: first argument must be array")))
	}

	fn := args[1]
	if fn.Type != value.TYPE_FUNCTION && fn.Type != value.TYPE_CLOSURE && fn.Type != value.TYPE_BUILTIN {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("filter: second argument must be function")))
	}

	elements := arr.AsArray()
	var results []value.Value
	for _, elem := range elements {
		result, err := vm.callValueSync(fn, []value.Value{elem})
		if err != nil {
			vm.sp = vm.sp - len(args) - 1
			return vm.push(value.Failure(value.String(fmt.Sprintf("filter: %v", err))))
		}
		if result.IsTruthy() {
			results = append(results, elem)
		}
	}

	vm.sp = vm.sp - len(args) - 1
	return vm.push(value.Array(results))
}

// executeReduce implements reduce(arr, fn, init) - reduces array to single value
// For pipeline: arr |> reduce(fn, init) -> reduce(arr, fn, init)
func (vm *VM) executeReduce(args []value.Value) error {
	if len(args) != 3 {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("reduce: expected 3 arguments (array, fn, init)")))
	}

	arr := args[0]
	if arr.Type != value.TYPE_ARRAY {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("reduce: first argument must be array")))
	}

	fn := args[1]
	if fn.Type != value.TYPE_FUNCTION && fn.Type != value.TYPE_CLOSURE && fn.Type != value.TYPE_BUILTIN {
		vm.sp = vm.sp - len(args) - 1
		return vm.push(value.Failure(value.String("reduce: second argument must be function")))
	}

	init := args[2]

	elements := arr.AsArray()
	accumulator := init
	for _, elem := range elements {
		result, err := vm.callValueSync(fn, []value.Value{accumulator, elem})
		if err != nil {
			vm.sp = vm.sp - len(args) - 1
			return vm.push(value.Failure(value.String(fmt.Sprintf("reduce: %v", err))))
		}
		accumulator = result
	}

	vm.sp = vm.sp - len(args) - 1
	return vm.push(accumulator)
}

// callValueSync calls a function or builtin synchronously and returns the result
func (vm *VM) callValueSync(callee value.Value, args []value.Value) (value.Value, error) {
	switch callee.Type {
	case value.TYPE_CLOSURE:
		return vm.callClosureSync(callee.AsClosure(), args)
	case value.TYPE_FUNCTION:
		// Wrap bare function in a closure for consistency
		cl := &value.Closure{Fn: callee.AsFunction()}
		return vm.callClosureSync(cl, args)
	case value.TYPE_BUILTIN:
		builtin := callee.AsBuiltin()
		if builtin.Fn == nil {
			return value.Null(), fmt.Errorf("builtin %s cannot be used as callback", builtin.Name)
		}
		return builtin.Fn(args...), nil
	case value.TYPE_PARTIAL_BUILTIN:
		partial := callee.AsPartialBuiltin()
		allArgs := append(partial.Args, args...)
		return vm.callPartialBuiltinSync(partial.Name, allArgs)
	default:
		return value.Null(), fmt.Errorf("cannot call %s", callee.Type)
	}
}

// callPartialBuiltinSync executes a partial builtin synchronously
func (vm *VM) callPartialBuiltinSync(name string, args []value.Value) (value.Value, error) {
	switch name {
	case "map":
		if len(args) != 2 {
			return value.Failure(value.String("map: expected 2 arguments")), nil
		}
		fn := args[0]
		arr := args[1]
		if arr.Type != value.TYPE_ARRAY {
			return value.Failure(value.String("map: second argument must be array")), nil
		}
		elements := arr.AsArray()
		results := make([]value.Value, len(elements))
		for i, elem := range elements {
			result, err := vm.callValueSync(fn, []value.Value{elem})
			if err != nil {
				return value.Failure(value.String(fmt.Sprintf("map: %v", err))), nil
			}
			results[i] = result
		}
		return value.Array(results), nil
	case "filter":
		if len(args) != 2 {
			return value.Failure(value.String("filter: expected 2 arguments")), nil
		}
		fn := args[0]
		arr := args[1]
		if arr.Type != value.TYPE_ARRAY {
			return value.Failure(value.String("filter: second argument must be array")), nil
		}
		elements := arr.AsArray()
		var results []value.Value
		for _, elem := range elements {
			result, err := vm.callValueSync(fn, []value.Value{elem})
			if err != nil {
				return value.Failure(value.String(fmt.Sprintf("filter: %v", err))), nil
			}
			if result.IsTruthy() {
				results = append(results, elem)
			}
		}
		return value.Array(results), nil
	case "reduce":
		if len(args) != 3 {
			return value.Failure(value.String("reduce: expected 3 arguments")), nil
		}
		fn := args[0]
		accumulator := args[1]
		arr := args[2]
		if arr.Type != value.TYPE_ARRAY {
			return value.Failure(value.String("reduce: third argument must be array")), nil
		}
		elements := arr.AsArray()
		for _, elem := range elements {
			result, err := vm.callValueSync(fn, []value.Value{accumulator, elem})
			if err != nil {
				return value.Failure(value.String(fmt.Sprintf("reduce: %v", err))), nil
			}
			accumulator = result
		}
		return accumulator, nil
	default:
		return value.Null(), fmt.Errorf("unknown partial builtin: %s", name)
	}
}

// callClosureSync executes a closure synchronously and returns its result
func (vm *VM) callClosureSync(cl *value.Closure, args []value.Value) (value.Value, error) {
	if len(args) != len(cl.Fn.Parameters) {
		return value.Null(), fmt.Errorf("wrong number of arguments: want=%d, got=%d",
			len(cl.Fn.Parameters), len(args))
	}

	// Save current state
	savedSP := vm.sp
	savedFramesIndex := vm.framesIndex

	// Set up the call: push closure then args onto stack
	vm.push(value.ClosureVal(cl))
	for _, arg := range args {
		vm.push(arg)
	}

	// Create new frame
	frame := NewFrame(cl, vm.sp-len(args))
	vm.pushFrame(frame)
	vm.sp = frame.basePointer + cl.Fn.NumLocals

	// The frame index for the closure we're executing
	targetFrameIndex := vm.framesIndex

	// Run until our frame returns
	for vm.framesIndex >= targetFrameIndex {
		currentFrame := vm.currentFrame()
		if currentFrame.ip >= len(currentFrame.Instructions())-1 {
			break
		}
		currentFrame.ip++

		ip := currentFrame.ip
		ins := currentFrame.Instructions()
		op := bytecode.OpCode(ins[ip])

		// Check for return from our target frame
		if op == bytecode.OpReturn && vm.framesIndex == targetFrameIndex {
			returnValue := vm.pop()
			vm.popFrame()

			// Restore state
			vm.sp = savedSP
			vm.framesIndex = savedFramesIndex

			return returnValue, nil
		}

		// Execute the opcode (including nested calls and returns)
		err := vm.executeOpcode(op)
		if err != nil {
			// Restore state on error
			vm.sp = savedSP
			vm.framesIndex = savedFramesIndex
			return value.Null(), err
		}
	}

	// Function ended without explicit return
	result := vm.StackTop()
	vm.sp = savedSP
	vm.framesIndex = savedFramesIndex
	return result, nil
}

// StackTop returns the top of the stack
func (vm *VM) StackTop() value.Value {
	if vm.sp == 0 {
		return value.Null()
	}
	return vm.stack[vm.sp-1]
}

// LastPoppedStackElem returns the last popped element
func (vm *VM) LastPoppedStackElem() value.Value {
	return vm.stack[vm.sp]
}

// Globals returns the globals slice
func (vm *VM) Globals() []value.Value {
	return vm.globals
}

// ============================================================
// Async execution functions
// ============================================================

// currentStay returns the current stay loop, or nil if not in a stay
func (vm *VM) currentStay() *StayLoop {
	if len(vm.stayStack) == 0 {
		return nil
	}
	return vm.stayStack[len(vm.stayStack)-1]
}

// executeStayBegin initializes a new stay loop with the given state
func (vm *VM) executeStayBegin(numPairs int) error {
	state := value.NewHash()

	// Pop key-value pairs from stack (in reverse order)
	pairs := make([]struct{ key, val value.Value }, numPairs)
	for i := numPairs - 1; i >= 0; i-- {
		pairs[i].val = vm.pop()
		pairs[i].key = vm.pop()
	}

	// Add pairs in original order
	for _, pair := range pairs {
		state.Set(pair.key, pair.val)
	}

	// Create new stay loop
	stay := &StayLoop{
		State:      state,
		Handlers:   []*value.Handler{},
		Tasks:      []*value.Task{},
		EventQueue: make(chan Event, 1024),
		Done:       make(chan value.Value, 1),
		ShouldExit: false,
	}

	// Push onto stay stack
	vm.stayStack = append(vm.stayStack, stay)

	return nil
}

// executeStayEnd runs the event loop for the current stay
func (vm *VM) executeStayEnd() error {
	stay := vm.currentStay()
	if stay == nil {
		return fmt.Errorf("async.stay: not in a stay block")
	}

	// Run the event loop
	result, err := vm.runStayLoop(stay)
	if err != nil {
		return err
	}

	// Pop stay from stack
	vm.stayStack = vm.stayStack[:len(vm.stayStack)-1]

	// Push result wrapped in success
	return vm.push(value.Success(result))
}

// runStayLoop executes the event loop for a stay block
func (vm *VM) runStayLoop(stay *StayLoop) (value.Value, error) {
	for !stay.ShouldExit {
		// Wait for events
		select {
		case event := <-stay.EventQueue:
			// Execute the handler callback synchronously
			if event.Handler.Status == value.HandlerActive {
				// EOF events don't pass a value to the callback (handler is () => ...)
				var args []value.Value
				if event.Handler.Source.Kind != value.EventSourceEOF {
					args = []value.Value{event.Value}
				}
				result, err := vm.callValueSync(event.Handler.Callback, args)
				if err != nil {
					return value.Null(), err
				}
				// The callback may call async.continue or async.leave
				// which modifies the stay state
				_ = result
			}

		case result := <-stay.Done:
			// async.leave was called
			return result, nil

		default:
			// Check if any active handlers exist
			hasActive := false
			for _, h := range stay.Handlers {
				if h.Status == value.HandlerActive {
					hasActive = true
					break
				}
			}
			// If no active handlers and no pending tasks, exit with null
			if !hasActive && len(stay.Tasks) == 0 {
				return value.Null(), nil
			}
			// Small sleep to avoid busy waiting
			time.Sleep(1 * time.Millisecond)
		}
	}

	return stay.LeaveValue, nil
}

// executeStayGetState gets a state value by key
func (vm *VM) executeStayGetState(key string) error {
	stay := vm.currentStay()
	if stay == nil {
		return fmt.Errorf("async.stay: not in a stay block")
	}

	val, found := stay.State.Get(value.String(key))
	if !found {
		return vm.push(value.Null())
	}
	return vm.push(val)
}

// executeContinue updates state and continues the stay loop
func (vm *VM) executeContinue(numPairs int) error {
	stay := vm.currentStay()
	if stay == nil {
		return fmt.Errorf("async.continue: not in a stay block")
	}

	// Pop key-value pairs from stack (in reverse order)
	pairs := make([]struct{ key, val value.Value }, numPairs)
	for i := numPairs - 1; i >= 0; i-- {
		pairs[i].val = vm.pop()
		pairs[i].key = vm.pop()
	}

	// Update state
	for _, pair := range pairs {
		stay.State.Set(pair.key, pair.val)
	}

	// Continue returns null (the loop continues)
	return vm.push(value.Null())
}

// executeLeave exits the stay loop with a value
func (vm *VM) executeLeave() error {
	stay := vm.currentStay()
	if stay == nil {
		return fmt.Errorf("async.leave: not in a stay block")
	}

	// Pop the leave value
	leaveValue := vm.pop()
	stay.LeaveValue = leaveValue
	stay.ShouldExit = true

	// Send to done channel (non-blocking)
	select {
	case stay.Done <- leaveValue:
	default:
	}

	// Return the leave value
	return vm.push(leaveValue)
}

// executeSpawn creates a new async task
func (vm *VM) executeSpawn() error {
	fn := vm.pop()
	if fn.Type != value.TYPE_CLOSURE && fn.Type != value.TYPE_FUNCTION && fn.Type != value.TYPE_BUILTIN {
		return fmt.Errorf("async.spawn: expected function, got %s", fn.Type)
	}

	taskID := atomic.AddInt64(&vm.taskCounter, 1)
	doneSource := &value.EventSource{
		Kind:    value.EventSourceTaskDone,
		Channel: make(chan value.Value, 1),
		Done:    make(chan struct{}),
	}

	task := &value.Task{
		ID:     taskID,
		Done:   doneSource,
		Cancel: make(chan struct{}),
	}
	task.SetStatus(value.TaskPending)

	// Register task with current stay loop
	if stay := vm.currentStay(); stay != nil {
		stay.Tasks = append(stay.Tasks, task)
	}

	// Start the task in a goroutine
	go func() {
		task.SetStatus(value.TaskRunning)

		select {
		case <-task.Cancel:
			task.SetStatus(value.TaskCancelled)
			task.SetResult(value.Null())
		default:
			result, err := vm.callValueSync(fn, []value.Value{})
			if err != nil {
				task.SetStatus(value.TaskFailed)
				task.SetError(err)
				task.SetResult(value.Failure(value.String(err.Error())))
			} else {
				task.SetStatus(value.TaskCompleted)
				task.SetResult(result)
			}
		}

		// Send result to done channel
		task.Done.Channel <- task.GetResult()
		close(task.Done.Done)
	}()

	return vm.push(value.TaskVal(task))
}

// executeExpects creates a handler for an event source
func (vm *VM) executeExpects() error {
	callback := vm.pop()
	source := vm.pop()

	if source.Type != value.TYPE_EVENT_SOURCE {
		return fmt.Errorf("async.expects: expected event_source, got %s", source.Type)
	}

	if callback.Type != value.TYPE_CLOSURE && callback.Type != value.TYPE_FUNCTION && callback.Type != value.TYPE_BUILTIN {
		return fmt.Errorf("async.expects: expected function, got %s", callback.Type)
	}

	handlerID := atomic.AddInt64(&vm.handlerCounter, 1)
	handler := &value.Handler{
		ID:       handlerID,
		Status:   value.HandlerDormant,
		Source:   source.AsEventSource(),
		Callback: callback,
	}

	// Register handler with current stay loop
	if stay := vm.currentStay(); stay != nil {
		stay.Handlers = append(stay.Handlers, handler)
	}

	return vm.push(value.HandlerVal(handler))
}

// executeHandlerReady activates a handler
func (vm *VM) executeHandlerReady() error {
	handlerVal := vm.pop()
	if handlerVal.Type != value.TYPE_HANDLER {
		return fmt.Errorf("handler.ready: expected handler, got %s", handlerVal.Type)
	}

	handler := handlerVal.AsHandler()
	handler.Status = value.HandlerActive

	stay := vm.currentStay()
	if stay == nil {
		return vm.push(handlerVal)
	}

	// Start listening for events from the source
	go func() {
		for {
			select {
			case val, ok := <-handler.Source.Channel:
				if !ok {
					return // Channel closed
				}
				if handler.Status == value.HandlerActive {
					stay.EventQueue <- Event{Handler: handler, Value: val}
				}
			case <-handler.Source.Done:
				return
			}
		}
	}()

	return vm.push(handlerVal)
}

// executeHandlerReset resets a handler
func (vm *VM) executeHandlerReset() error {
	handlerVal := vm.pop()
	if handlerVal.Type != value.TYPE_HANDLER {
		return fmt.Errorf("handler.reset: expected handler, got %s", handlerVal.Type)
	}

	handler := handlerVal.AsHandler()
	handler.Status = value.HandlerDormant

	return vm.push(handlerVal)
}

// executeHandlerPause pauses a handler
func (vm *VM) executeHandlerPause() error {
	handlerVal := vm.pop()
	if handlerVal.Type != value.TYPE_HANDLER {
		return fmt.Errorf("handler.pause: expected handler, got %s", handlerVal.Type)
	}

	handler := handlerVal.AsHandler()
	handler.Status = value.HandlerPaused

	return vm.push(handlerVal)
}

// executeHandlerResume resumes a paused handler
func (vm *VM) executeHandlerResume() error {
	handlerVal := vm.pop()
	if handlerVal.Type != value.TYPE_HANDLER {
		return fmt.Errorf("handler.resume: expected handler, got %s", handlerVal.Type)
	}

	handler := handlerVal.AsHandler()
	if handler.Status == value.HandlerPaused {
		handler.Status = value.HandlerActive
	}

	return vm.push(handlerVal)
}

// executeCancel cancels a task or handler
func (vm *VM) executeCancel() error {
	target := vm.pop()

	switch target.Type {
	case value.TYPE_TASK:
		task := target.AsTask()
		close(task.Cancel)
		task.SetStatus(value.TaskCancelled)
	case value.TYPE_HANDLER:
		handler := target.AsHandler()
		handler.Status = value.HandlerCancelled
	default:
		return fmt.Errorf("async.cancel: expected task or handler, got %s", target.Type)
	}

	return vm.push(value.Null())
}

// executeAll waits for all tasks to complete
func (vm *VM) executeAll() error {
	tasksVal := vm.pop()
	if tasksVal.Type != value.TYPE_ARRAY {
		return fmt.Errorf("async.all: expected array of tasks, got %s", tasksVal.Type)
	}

	tasks := tasksVal.AsArray()
	results := make([]value.Value, len(tasks))

	for i, taskVal := range tasks {
		if taskVal.Type != value.TYPE_TASK {
			results[i] = value.Failure(value.String("async.all: array contains non-task"))
			continue
		}

		task := taskVal.AsTask()
		// Wait for task to complete
		result := <-task.Done.Channel
		results[i] = result
	}

	return vm.push(value.Array(results))
}

// executeTimeout creates a timeout event source
func (vm *VM) executeTimeout() error {
	msVal := vm.pop()
	if msVal.Type != value.TYPE_INT {
		return fmt.Errorf("schedule.timeout: expected integer, got %s", msVal.Type)
	}

	ms := msVal.AsInt()
	es := &value.EventSource{
		Kind:    value.EventSourceTimeout,
		Channel: make(chan value.Value, 1),
		Done:    make(chan struct{}),
	}

	go func() {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		select {
		case es.Channel <- value.Null():
		case <-es.Done:
		}
		close(es.Done)
	}()

	return vm.push(value.EventSourceVal(es))
}

// executeInterval creates an interval event source
func (vm *VM) executeInterval() error {
	msVal := vm.pop()
	if msVal.Type != value.TYPE_INT {
		return fmt.Errorf("schedule.interval: expected integer, got %s", msVal.Type)
	}

	ms := msVal.AsInt()
	es := &value.EventSource{
		Kind:    value.EventSourceInterval,
		Channel: make(chan value.Value, 10),
		Done:    make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case es.Channel <- value.Null():
				default:
					// Channel full, skip this tick
				}
			case <-es.Done:
				return
			}
		}
	}()

	return vm.push(value.EventSourceVal(es))
}

// initStdinSource initializes the shared stdin event source
func (vm *VM) initStdinSource() {
	vm.stdinOnce.Do(func() {
		vm.stdinSource = &value.EventSource{
			Kind:    value.EventSourceStdin,
			Channel: make(chan value.Value, 100),
			Done:    make(chan struct{}),
		}
		vm.eofSource = &value.EventSource{
			Kind:    value.EventSourceEOF,
			Channel: make(chan value.Value, 1),
			Done:    make(chan struct{}),
		}

		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				select {
				case vm.stdinSource.Channel <- value.String(line):
				case <-vm.stdinSource.Done:
					return
				}
			}
			// EOF reached - send null value before closing to trigger eof event handler
			vm.eofSource.Channel <- value.Null()
			close(vm.eofSource.Channel)
		}()
	})
}

// Built-in functions

func builtinLen(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("len: expected 1 argument"))
	}
	switch args[0].Type {
	case value.TYPE_STRING:
		return value.Int(int64(len(args[0].AsString())))
	case value.TYPE_ARRAY:
		return value.Int(int64(len(args[0].AsArray())))
	case value.TYPE_TUPLE:
		return value.Int(int64(len(args[0].AsTuple())))
	case value.TYPE_HASH:
		return value.Int(int64(len(args[0].AsHash().Pairs)))
	default:
		return value.Failure(value.String(fmt.Sprintf("len: unsupported type %s", args[0].Type)))
	}
}

func builtinConcat(args ...value.Value) value.Value {
	if len(args) == 0 {
		return value.String("")
	}
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
		var sb strings.Builder
		for _, arg := range args {
			sb.WriteString(arg.String())
		}
		return value.String(sb.String())
	}
}

func builtinToString(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("to_string: expected 1 argument"))
	}
	return value.String(args[0].String())
}

func builtinGet(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 3 {
		return value.Failure(value.String("get: expected 2 or 3 arguments"))
	}

	// Handle array: get(array, index, [default])
	if args[0].Type == value.TYPE_ARRAY {
		if args[1].Type != value.TYPE_INT {
			return value.Failure(value.String("get: index must be integer for array"))
		}
		arr := args[0].AsArray()
		idx := int(args[1].AsInt())
		if idx < 0 || idx >= len(arr) {
			if len(args) == 3 {
				return args[2]
			}
			return value.Null()
		}
		return arr[idx]
	}

	// Handle hash: get(hash, key, [default])
	if args[0].Type == value.TYPE_HASH {
		if args[1].Type != value.TYPE_STRING {
			return value.Failure(value.String("get: key must be string for hash"))
		}
		hash := args[0].AsHash()
		key := args[1].AsString()
		if idx, ok := hash.Lookup[key]; ok {
			return hash.Pairs[idx].Value
		}
		if len(args) == 3 {
			return args[2]
		}
		return value.Null()
	}

	return value.Failure(value.String("get: first argument must be array or hash"))
}

func builtinHas(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("has: expected 2 arguments"))
	}

	// Handle array: has(array, element) - check if element exists
	if args[0].Type == value.TYPE_ARRAY {
		arr := args[0].AsArray()
		for _, v := range arr {
			if v.Equals(args[1]) {
				return value.Bool(true)
			}
		}
		return value.Bool(false)
	}

	// Handle hash: has(hash, key) - check if key exists
	if args[0].Type == value.TYPE_HASH {
		if args[1].Type != value.TYPE_STRING {
			return value.Failure(value.String("has: key must be string for hash"))
		}
		hash := args[0].AsHash()
		key := args[1].AsString()
		_, exists := hash.Lookup[key]
		return value.Bool(exists)
	}

	return value.Bool(false)
}

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

func builtinRange(args ...value.Value) value.Value {
	var start, step int64 = 0, 1
	var end int64
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

func builtinKeys(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("keys: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String(fmt.Sprintf("keys: expected hash, got %s", args[0].Type)))
	}
	hash := args[0].AsHash()
	return value.Array(hash.Keys())
}

func builtinValues(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("values: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String(fmt.Sprintf("values: expected hash, got %s", args[0].Type)))
	}
	hash := args[0].AsHash()
	return value.Array(hash.Values())
}

// hash_set creates a new hash with the given key-value pair added/updated
// hash_set(hash, key, value) -> new_hash
func builtinHashSet(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("hash_set: expected 3 arguments (hash, key, value)"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String(fmt.Sprintf("hash_set: expected hash as first argument, got %s", args[0].Type)))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String(fmt.Sprintf("hash_set: expected string as key, got %s", args[1].Type)))
	}
	hash := args[0].AsHash()
	key := args[1].AsString()
	val := args[2]

	// Create a new hash with all existing pairs plus the new one
	newPairs := make([]value.HashPair, 0, len(hash.Pairs)+1)
	newLookup := make(map[string]int)
	keyFound := false
	for i, pair := range hash.Pairs {
		pairKey := pair.Key.AsString()
		if pairKey == key {
			// Replace existing key
			newPairs = append(newPairs, value.HashPair{Key: pair.Key, Value: val})
			newLookup[pairKey] = len(newPairs) - 1
			keyFound = true
		} else {
			newPairs = append(newPairs, pair)
			newLookup[pairKey] = len(newPairs) - 1
		}
		_ = i // suppress unused variable warning
	}
	if !keyFound {
		newPairs = append(newPairs, value.HashPair{Key: value.String(key), Value: val})
		newLookup[key] = len(newPairs) - 1
	}
	return value.HashVal(&value.Hash{Pairs: newPairs, Lookup: newLookup})
}

// hash_merge merges two hashes, with the second hash's values taking precedence
// hash_merge(hash1, hash2) -> new_hash
func builtinHashMerge(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("hash_merge: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_HASH {
		return value.Failure(value.String(fmt.Sprintf("hash_merge: expected hash as first argument, got %s", args[0].Type)))
	}
	if args[1].Type != value.TYPE_HASH {
		return value.Failure(value.String(fmt.Sprintf("hash_merge: expected hash as second argument, got %s", args[1].Type)))
	}
	hash1 := args[0].AsHash()
	hash2 := args[1].AsHash()

	// Create a map for efficient lookup
	merged := make(map[string]value.Value)
	for _, pair := range hash1.Pairs {
		merged[pair.Key.AsString()] = pair.Value
	}
	for _, pair := range hash2.Pairs {
		merged[pair.Key.AsString()] = pair.Value
	}

	// Convert back to pairs with lookup
	newPairs := make([]value.HashPair, 0, len(merged))
	newLookup := make(map[string]int)
	for k, v := range merged {
		newPairs = append(newPairs, value.HashPair{Key: value.String(k), Value: v})
		newLookup[k] = len(newPairs) - 1
	}
	return value.HashVal(&value.Hash{Pairs: newPairs, Lookup: newLookup})
}

// Module functions

// print - prints values without newline (core.io!)
func (vm *VM) builtinPrint(args ...value.Value) value.Value {
	w := vm.getOutput()
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprint(w, arg.String())
	}
	return value.Null()
}

// println - prints values with newline (core.io!)
func (vm *VM) builtinPrintln(args ...value.Value) value.Value {
	w := vm.getOutput()
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprint(w, arg.String())
	}
	fmt.Fprintln(w)
	return value.Null()
}

// ============================================================
// core.io functions
// ============================================================

// read_file - reads entire file as string (core.io!)
func builtinReadFile(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("read_file: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("read_file: path must be string"))
	}
	content, err := os.ReadFile(args[0].AsString())
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("read_file: %v", err)))
	}
	return value.Success(value.String(string(content)))
}

// write_file - writes string to file (core.io!)
func builtinWriteFile(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("write_file: expected 2 arguments (path, content)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("write_file: path must be string"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("write_file: content must be string"))
	}
	err := os.WriteFile(args[0].AsString(), []byte(args[1].AsString()), 0644)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("write_file: %v", err)))
	}
	return value.Success(value.Null())
}

// list_dir - lists files/directories in a directory (core.io)
func builtinListDir(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("list_dir: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("list_dir: path must be string"))
	}
	entries, err := os.ReadDir(args[0].AsString())
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("list_dir: %v", err)))
	}
	result := make([]value.Value, len(entries))
	for i, entry := range entries {
		result[i] = value.String(entry.Name())
	}
	return value.Success(value.Array(result))
}

// mkdir - creates a directory (and parents) (core.io!)
func builtinMkdir(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("mkdir: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("mkdir: path must be string"))
	}
	err := os.MkdirAll(args[0].AsString(), 0755)
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("mkdir: %v", err)))
	}
	return value.Success(value.Null())
}

// exists - checks if a file or directory exists (core.io)
func builtinExists(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("exists: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("exists: path must be string"))
	}
	_, err := os.Stat(args[0].AsString())
	if err == nil {
		return value.Bool(true)
	}
	if os.IsNotExist(err) {
		return value.Bool(false)
	}
	return value.Failure(value.String(fmt.Sprintf("exists: %v", err)))
}

// delete_file - deletes a file (core.io!)
func builtinDeleteFile(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("delete_file: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("delete_file: path must be string"))
	}
	err := os.Remove(args[0].AsString())
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("delete_file: %v", err)))
	}
	return value.Success(value.Null())
}

// file_info - returns file metadata as a hash (core.io)
func builtinFileInfo(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("file_info: expected 1 argument (path)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("file_info: path must be string"))
	}
	info, err := os.Stat(args[0].AsString())
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("file_info: %v", err)))
	}
	h := value.NewHash()
	h.Set(value.String("name"), value.String(info.Name()))
	h.Set(value.String("size"), value.Int(info.Size()))
	h.Set(value.String("is_dir"), value.Bool(info.IsDir()))
	h.Set(value.String("modified"), value.String(info.ModTime().UTC().Format("2006-01-02T15:04:05Z")))
	return value.Success(value.HashVal(h))
}

// format - formats a string with arguments (core.io)
func builtinFormat(args ...value.Value) value.Value {
	if len(args) < 1 {
		return value.Failure(value.String("format: expected at least 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("format: first argument must be string"))
	}
	format := args[0].AsString()
	// Simple placeholder replacement: {} -> value
	result := format
	for i := 1; i < len(args); i++ {
		placeholder := "{}"
		idx := strings.Index(result, placeholder)
		if idx == -1 {
			break
		}
		result = result[:idx] + args[i].String() + result[idx+len(placeholder):]
	}
	return value.String(result)
}

// ============================================================
// core.string functions
// ============================================================

// split - splits string by separator (core.string)
func builtinSplit(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("split: expected 2 arguments (string, separator)"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("split: arguments must be strings"))
	}
	parts := strings.Split(args[0].AsString(), args[1].AsString())
	result := make([]value.Value, len(parts))
	for i, p := range parts {
		result[i] = value.String(p)
	}
	return value.Array(result)
}

// join - joins array with separator (core.string)
func builtinJoin(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("join: expected 2 arguments (array, separator)"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("join: first argument must be array"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("join: separator must be string"))
	}
	arr := args[0].AsArray()
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = v.String()
	}
	return value.String(strings.Join(parts, args[1].AsString()))
}

// trim - removes whitespace from both ends (core.string)
func builtinTrim(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("trim: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("trim: argument must be string"))
	}
	return value.String(strings.TrimSpace(args[0].AsString()))
}

// upper - converts to uppercase (core.string)
func builtinUpper(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("upper: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("upper: argument must be string"))
	}
	return value.String(strings.ToUpper(args[0].AsString()))
}

// lower - converts to lowercase (core.string)
func builtinLower(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("lower: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("lower: argument must be string"))
	}
	return value.String(strings.ToLower(args[0].AsString()))
}

// starts_with - checks if string starts with prefix (core.string)
func builtinStartsWith(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("starts_with: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("starts_with: arguments must be strings"))
	}
	return value.Bool(strings.HasPrefix(args[0].AsString(), args[1].AsString()))
}

// ends_with - checks if string ends with suffix (core.string)
func builtinEndsWith(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("ends_with: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("ends_with: arguments must be strings"))
	}
	return value.Bool(strings.HasSuffix(args[0].AsString(), args[1].AsString()))
}

// contains - checks if string contains substring (core.string)
func builtinContains(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("contains: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("contains: arguments must be strings"))
	}
	return value.Bool(strings.Contains(args[0].AsString(), args[1].AsString()))
}

// replace - replaces all occurrences (core.string)
func builtinReplace(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("replace: expected 3 arguments (string, old, new)"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING || args[2].Type != value.TYPE_STRING {
		return value.Failure(value.String("replace: arguments must be strings"))
	}
	return value.String(strings.ReplaceAll(args[0].AsString(), args[1].AsString(), args[2].AsString()))
}

// substring - extracts substring (core.string)
func builtinSubstring(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 3 {
		return value.Failure(value.String("substring: expected 2 or 3 arguments (string, start, [end])"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("substring: first argument must be string"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("substring: start must be integer"))
	}
	s := args[0].AsString()
	start := int(args[1].AsInt())
	if start < 0 {
		start = 0
	}
	if start >= len(s) {
		return value.String("")
	}
	end := len(s)
	if len(args) == 3 {
		if args[2].Type != value.TYPE_INT {
			return value.Failure(value.String("substring: end must be integer"))
		}
		end = int(args[2].AsInt())
		if end > len(s) {
			end = len(s)
		}
	}
	if start >= end {
		return value.String("")
	}
	return value.String(s[start:end])
}

// char_at - gets character at index (core.string)
func builtinCharAt(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("char_at: expected 2 arguments (string, index)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("char_at: first argument must be string"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("char_at: index must be integer"))
	}
	s := args[0].AsString()
	idx := int(args[1].AsInt())
	if idx < 0 || idx >= len(s) {
		return value.Null()
	}
	return value.String(string(s[idx]))
}

// index_of - finds first index of substring (core.string)
func builtinStringIndexOf(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("index_of: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_STRING || args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("index_of: arguments must be strings"))
	}
	idx := strings.Index(args[0].AsString(), args[1].AsString())
	return value.Int(int64(idx))
}

// ============================================================
// core.array functions
// ============================================================

// reverse - reverses array (core.array)
func builtinReverse(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("reverse: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("reverse: argument must be array"))
	}
	arr := args[0].AsArray()
	result := make([]value.Value, len(arr))
	for i := 0; i < len(arr); i++ {
		result[i] = arr[len(arr)-1-i]
	}
	return value.Array(result)
}

// slice - extracts slice of array (core.array)
func builtinSlice(args ...value.Value) value.Value {
	if len(args) < 2 || len(args) > 3 {
		return value.Failure(value.String("slice: expected 2 or 3 arguments (array, start, [end])"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("slice: first argument must be array"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("slice: start must be integer"))
	}
	arr := args[0].AsArray()
	start := int(args[1].AsInt())
	if start < 0 {
		start = 0
	}
	if start >= len(arr) {
		return value.Array([]value.Value{})
	}
	end := len(arr)
	if len(args) == 3 {
		if args[2].Type != value.TYPE_INT {
			return value.Failure(value.String("slice: end must be integer"))
		}
		end = int(args[2].AsInt())
		if end > len(arr) {
			end = len(arr)
		}
	}
	if start >= end {
		return value.Array([]value.Value{})
	}
	result := make([]value.Value, end-start)
	copy(result, arr[start:end])
	return value.Array(result)
}

// index_of - finds first index of value in array (core.array)
func builtinArrayIndexOf(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("index_of: expected 2 arguments (array, value)"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("index_of: first argument must be array"))
	}
	arr := args[0].AsArray()
	for i, v := range arr {
		if v.Equals(args[1]) {
			return value.Int(int64(i))
		}
	}
	return value.Int(-1)
}

// flatten - flattens nested arrays one level (core.array)
func builtinFlatten(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("flatten: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("flatten: argument must be array"))
	}
	arr := args[0].AsArray()
	var result []value.Value
	for _, v := range arr {
		if v.Type == value.TYPE_ARRAY {
			result = append(result, v.AsArray()...)
		} else {
			result = append(result, v)
		}
	}
	return value.Array(result)
}

// unique - removes duplicates from array (core.array)
func builtinUnique(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("unique: expected 1 argument"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("unique: argument must be array"))
	}
	arr := args[0].AsArray()
	var result []value.Value
	for _, v := range arr {
		found := false
		for _, r := range result {
			if v.Equals(r) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, v)
		}
	}
	return value.Array(result)
}

// zip - combines two arrays into array of pairs (core.array)
func builtinZip(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("zip: expected 2 arguments"))
	}
	if args[0].Type != value.TYPE_ARRAY || args[1].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("zip: arguments must be arrays"))
	}
	arr1 := args[0].AsArray()
	arr2 := args[1].AsArray()
	minLen := len(arr1)
	if len(arr2) < minLen {
		minLen = len(arr2)
	}
	result := make([]value.Value, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = value.Array([]value.Value{arr1[i], arr2[i]})
	}
	return value.Array(result)
}

// take - takes first n elements (core.array)
func builtinTake(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("take: expected 2 arguments (array, count)"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("take: first argument must be array"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("take: count must be integer"))
	}
	arr := args[0].AsArray()
	n := int(args[1].AsInt())
	if n <= 0 {
		return value.Array([]value.Value{})
	}
	if n >= len(arr) {
		result := make([]value.Value, len(arr))
		copy(result, arr)
		return value.Array(result)
	}
	result := make([]value.Value, n)
	copy(result, arr[:n])
	return value.Array(result)
}

// drop - drops first n elements (core.array)
func builtinDrop(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("drop: expected 2 arguments (array, count)"))
	}
	if args[0].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("drop: first argument must be array"))
	}
	if args[1].Type != value.TYPE_INT {
		return value.Failure(value.String("drop: count must be integer"))
	}
	arr := args[0].AsArray()
	n := int(args[1].AsInt())
	if n <= 0 {
		result := make([]value.Value, len(arr))
		copy(result, arr)
		return value.Array(result)
	}
	if n >= len(arr) {
		return value.Array([]value.Value{})
	}
	result := make([]value.Value, len(arr)-n)
	copy(result, arr[n:])
	return value.Array(result)
}

// ============================================================
// core.async functions
// ============================================================

// builtinAsyncSpawn - async.spawn(fn) creates a new async task
func (vm *VM) builtinAsyncSpawn(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("async.spawn: expected 1 argument"))
	}

	fn := args[0]
	if fn.Type != value.TYPE_CLOSURE && fn.Type != value.TYPE_FUNCTION && fn.Type != value.TYPE_BUILTIN {
		return value.Failure(value.String(fmt.Sprintf("async.spawn: expected function, got %s", fn.Type)))
	}

	taskID := atomic.AddInt64(&vm.taskCounter, 1)
	doneSource := &value.EventSource{
		Kind:    value.EventSourceTaskDone,
		Channel: make(chan value.Value, 1),
		Done:    make(chan struct{}),
	}

	task := &value.Task{
		ID:     taskID,
		Done:   doneSource,
		Cancel: make(chan struct{}),
	}
	task.SetStatus(value.TaskPending)

	// Register task with current stay loop
	if stay := vm.currentStay(); stay != nil {
		vm.mu.Lock()
		stay.Tasks = append(stay.Tasks, task)
		vm.mu.Unlock()
	}

	// Clone VM for independent execution in goroutine
	spawnVM := vm.CloneForSpawn()

	// Start the task in a goroutine with cloned VM
	go func() {
		task.SetStatus(value.TaskRunning)

		select {
		case <-task.Cancel:
			task.SetStatus(value.TaskCancelled)
			task.SetResult(value.Null())
		default:
			result, err := spawnVM.callValueSync(fn, []value.Value{})
			if err != nil {
				task.SetStatus(value.TaskFailed)
				task.SetError(err)
				task.SetResult(value.Failure(value.String(err.Error())))
			} else {
				task.SetStatus(value.TaskCompleted)
				task.SetResult(result)
			}
		}

		// Send result to done channel
		select {
		case task.Done.Channel <- task.GetResult():
		default:
		}
		close(task.Done.Done)
	}()

	return value.TaskVal(task)
}

// builtinAsyncExpects - async.expects(callback) or async.expects(source, callback) creates an event handler
// Supports partial application: async.expects(callback) returns source => handler
func (vm *VM) builtinAsyncExpects(args ...value.Value) value.Value {
	if len(args) == 0 || len(args) > 2 {
		return value.Failure(value.String("async.expects: expected 1 or 2 arguments"))
	}

	// Partial application: async.expects(callback) returns function taking source
	if len(args) == 1 {
		callback := args[0]
		if callback.Type != value.TYPE_CLOSURE && callback.Type != value.TYPE_FUNCTION && callback.Type != value.TYPE_BUILTIN {
			return value.Failure(value.String(fmt.Sprintf("async.expects: expected function, got %s", callback.Type)))
		}
		// Return a partial builtin that will complete with the source
		return value.PartialBuiltinVal(&value.PartialBuiltin{
			Name: "async.expects",
			Args: []value.Value{callback},
		})
	}

	// Full application: async.expects(source, callback) or from pipe: source |> async.expects(callback)
	// When called via pipe, the order is: source first (piped), callback second (argument)
	// But with partial application, args are: [callback, source]
	source := args[0]
	callback := args[1]

	// If source is a function, swap (this happens when called as source |> async.expects(callback))
	if source.Type == value.TYPE_CLOSURE || source.Type == value.TYPE_FUNCTION || source.Type == value.TYPE_BUILTIN {
		source, callback = callback, source
	}

	if source.Type != value.TYPE_EVENT_SOURCE {
		return value.Failure(value.String(fmt.Sprintf("async.expects: expected event_source, got %s", source.Type)))
	}

	if callback.Type != value.TYPE_CLOSURE && callback.Type != value.TYPE_FUNCTION && callback.Type != value.TYPE_BUILTIN {
		return value.Failure(value.String(fmt.Sprintf("async.expects: expected function, got %s", callback.Type)))
	}

	handlerID := atomic.AddInt64(&vm.handlerCounter, 1)
	handler := &value.Handler{
		ID:       handlerID,
		Status:   value.HandlerDormant,
		Source:   source.AsEventSource(),
		Callback: callback,
	}

	// Register handler with current stay loop
	if stay := vm.currentStay(); stay != nil {
		vm.mu.Lock()
		stay.Handlers = append(stay.Handlers, handler)
		vm.mu.Unlock()
	}

	return value.HandlerVal(handler)
}

// builtinAsyncContinue - async.continue(state_updates) continues the stay loop with new state
func (vm *VM) builtinAsyncContinue(args ...value.Value) value.Value {
	stay := vm.currentStay()
	if stay == nil {
		return value.Failure(value.String("async.continue: not in a stay block"))
	}

	// args is a hash of state updates
	if len(args) == 1 && args[0].Type == value.TYPE_HASH {
		updates := args[0].AsHash()
		for _, pair := range updates.Pairs {
			stay.State.Set(pair.Key, pair.Value)
		}
	}

	return value.Null()
}

// builtinAsyncLeave - async.leave(value) exits the stay loop with a value
func (vm *VM) builtinAsyncLeave(args ...value.Value) value.Value {
	stay := vm.currentStay()
	if stay == nil {
		return value.Failure(value.String("async.leave: not in a stay block"))
	}

	var leaveValue value.Value = value.Null()
	if len(args) > 0 {
		leaveValue = args[0]
	}

	stay.LeaveValue = leaveValue
	stay.ShouldExit = true

	// Send to done channel (non-blocking)
	select {
	case stay.Done <- leaveValue:
	default:
	}

	return leaveValue
}

// builtinAsyncCancel - async.cancel(target) cancels a task or handler
func (vm *VM) builtinAsyncCancel(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("async.cancel: expected 1 argument"))
	}

	target := args[0]

	switch target.Type {
	case value.TYPE_TASK:
		task := target.AsTask()
		select {
		case <-task.Cancel:
			// Already cancelled
		default:
			close(task.Cancel)
		}
		task.SetStatus(value.TaskCancelled)
	case value.TYPE_HANDLER:
		handler := target.AsHandler()
		handler.Status = value.HandlerCancelled
	default:
		return value.Failure(value.String(fmt.Sprintf("async.cancel: expected task or handler, got %s", target.Type)))
	}

	return value.Null()
}

// builtinAsyncAll - async.all(tasks) waits for all tasks to complete
func (vm *VM) builtinAsyncAll(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("async.all: expected 1 argument"))
	}

	tasksVal := args[0]
	if tasksVal.Type != value.TYPE_ARRAY {
		return value.Failure(value.String(fmt.Sprintf("async.all: expected array of tasks, got %s", tasksVal.Type)))
	}

	tasks := tasksVal.AsArray()
	results := make([]value.Value, len(tasks))

	for i, taskVal := range tasks {
		if taskVal.Type != value.TYPE_TASK {
			results[i] = value.Failure(value.String("async.all: array contains non-task"))
			continue
		}

		task := taskVal.AsTask()
		// Wait for task to complete
		select {
		case result := <-task.Done.Channel:
			results[i] = result
		case <-time.After(60 * time.Second):
			results[i] = value.Failure(value.String("async.all: task timeout"))
		}
	}

	return value.Array(results)
}

// builtinAsyncChannel - async.channel() or async.channel(capacity) creates a message-passing channel
// Usage:
//
//	ch = async.channel()        // unbuffered channel
//	ch = async.channel(10)      // buffered channel with capacity 10
//
// Channel supports:
//
//	ch.send(value)              // send a value (returns null or failure)
//	ch.receive()                // receive a value (blocks until available)
//	ch.close()                  // close the channel
//	ch.source                   // EventSource for use with async.expects
//	ch.status                   // "open" or "closed"
func (vm *VM) builtinAsyncChannel(args ...value.Value) value.Value {
	capacity := 0

	if len(args) > 1 {
		return value.Failure(value.String("async.channel: expected 0 or 1 arguments"))
	}
	if len(args) == 1 {
		if args[0].Type != value.TYPE_INT {
			return value.Failure(value.String(fmt.Sprintf("async.channel: capacity must be an integer, got %s", args[0].Type)))
		}
		capacity = int(args[0].AsInt())
		if capacity < 0 {
			return value.Failure(value.String("async.channel: capacity must be non-negative"))
		}
	}

	ch := value.NewChannel(capacity)
	return value.ChannelVal(ch)
}

// ============================================================
// core.schedule functions
// ============================================================

// builtinScheduleTimeout - schedule.timeout(ms) creates a timeout event source
func (vm *VM) builtinScheduleTimeout(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("schedule.timeout: expected 1 argument"))
	}

	msVal := args[0]
	if msVal.Type != value.TYPE_INT {
		return value.Failure(value.String(fmt.Sprintf("schedule.timeout: expected integer, got %s", msVal.Type)))
	}

	ms := msVal.AsInt()
	es := &value.EventSource{
		Kind:    value.EventSourceTimeout,
		Channel: make(chan value.Value, 1),
		Done:    make(chan struct{}),
	}

	go func() {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		select {
		case es.Channel <- value.Null():
		case <-es.Done:
		}
	}()

	return value.EventSourceVal(es)
}

// builtinScheduleInterval - schedule.interval(ms) creates an interval event source
func (vm *VM) builtinScheduleInterval(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("schedule.interval: expected 1 argument"))
	}

	msVal := args[0]
	if msVal.Type != value.TYPE_INT {
		return value.Failure(value.String(fmt.Sprintf("schedule.interval: expected integer, got %s", msVal.Type)))
	}

	ms := msVal.AsInt()
	es := &value.EventSource{
		Kind:    value.EventSourceInterval,
		Channel: make(chan value.Value, 10),
		Done:    make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case es.Channel <- value.Null():
				default:
					// Channel full, skip this tick
				}
			case <-es.Done:
				return
			}
		}
	}()

	return value.EventSourceVal(es)
}

// ============================================================
// Handler methods (called via member access)
// ============================================================

// builtinHandlerReady - handler.ready() activates a handler
func (vm *VM) builtinHandlerReady(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("handler.ready: expected 1 argument (handler)"))
	}

	handlerVal := args[0]
	if handlerVal.Type != value.TYPE_HANDLER {
		return value.Failure(value.String(fmt.Sprintf("handler.ready: expected handler, got %s", handlerVal.Type)))
	}

	handler := handlerVal.AsHandler()
	handler.Status = value.HandlerActive

	stay := vm.currentStay()
	if stay == nil {
		return handlerVal
	}

	// Start listening for events from the source
	go func() {
		for {
			select {
			case val, ok := <-handler.Source.Channel:
				if !ok {
					return // Channel closed
				}
				if handler.Status == value.HandlerActive {
					select {
					case stay.EventQueue <- Event{Handler: handler, Value: val}:
					default:
						// Event queue full
					}
				}
			case <-handler.Source.Done:
				return
			}
		}
	}()

	return handlerVal
}

// Regex builtins

// builtinMatches checks if a string matches a regex pattern
// matches(string, regex) -> bool
func builtinMatches(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("matches: expected 2 arguments (string, regex)"))
	}

	// First argument must be a string
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("matches: first argument must be string"))
	}
	str := args[0].AsString()

	// Second argument must be a regex
	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("matches: second argument must be regex"))
	}
	re := args[1].AsRegex()

	// Perform the match using the pre-compiled regex
	matched := re.Re.MatchString(str)
	return value.Bool(matched)
}

// builtinFind finds the first match of a regex in a string
// find(string, regex) -> success(match) | failure("no match")
func builtinFind(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("find: expected 2 arguments (string, regex)"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("find: first argument must be string"))
	}
	str := args[0].AsString()

	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("find: second argument must be regex"))
	}
	re := args[1].AsRegex()

	match := re.Re.FindString(str)
	if match == "" {
		return value.Failure(value.String("no match"))
	}
	return value.Success(value.String(match))
}

// builtinFindAll finds all matches of a regex in a string
// find_all(string, regex) -> [match1, match2, ...]
func builtinFindAll(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("find_all: expected 2 arguments (string, regex)"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("find_all: first argument must be string"))
	}
	str := args[0].AsString()

	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("find_all: second argument must be regex"))
	}
	re := args[1].AsRegex()

	matches := re.Re.FindAllString(str, -1)
	result := make([]value.Value, len(matches))
	for i, m := range matches {
		result[i] = value.String(m)
	}
	return value.Array(result)
}

// builtinRegexReplace replaces all matches of a regex with a replacement string
// regex_replace(string, regex, replacement) -> string
func builtinRegexReplace(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("regex_replace: expected 3 arguments (string, regex, replacement)"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("regex_replace: first argument must be string"))
	}
	str := args[0].AsString()

	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("regex_replace: second argument must be regex"))
	}
	re := args[1].AsRegex()

	if args[2].Type != value.TYPE_STRING {
		return value.Failure(value.String("regex_replace: third argument must be string"))
	}
	replacement := args[2].AsString()

	result := re.Re.ReplaceAllString(str, replacement)
	return value.String(result)
}

// builtinRegexSplit splits a string by a regex pattern
// regex_split(string, regex) -> [part1, part2, ...]
func builtinRegexSplit(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("regex_split: expected 2 arguments (string, regex)"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("regex_split: first argument must be string"))
	}
	str := args[0].AsString()

	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("regex_split: second argument must be regex"))
	}
	re := args[1].AsRegex()

	parts := re.Re.Split(str, -1)
	result := make([]value.Value, len(parts))
	for i, p := range parts {
		result[i] = value.String(p)
	}
	return value.Array(result)
}

// builtinCapture returns capture groups from a regex match
// capture(string, regex) -> success([group0, group1, ...]) | failure("no match")
func builtinCapture(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("capture: expected 2 arguments (string, regex)"))
	}

	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("capture: first argument must be string"))
	}
	str := args[0].AsString()

	if args[1].Type != value.TYPE_REGEX {
		return value.Failure(value.String("capture: second argument must be regex"))
	}
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

// suggestSimilarName returns a "did you mean" suggestion if a similar name is found
func suggestSimilarName(name string, candidates []string) string {
	bestMatch := ""
	bestDist := 3 // max edit distance

	for _, c := range candidates {
		dist := editDistance(name, c)
		if dist < bestDist {
			bestDist = dist
			bestMatch = c
		}
	}

	if bestMatch != "" {
		return fmt.Sprintf(" (did you mean '%s'?)", bestMatch)
	}
	return ""
}

// editDistance computes the Levenshtein distance between two strings
func editDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := matrix[i-1][j] + 1
			ins := matrix[i][j-1] + 1
			sub := matrix[i-1][j-1] + cost
			min := del
			if ins < min {
				min = ins
			}
			if sub < min {
				min = sub
			}
			matrix[i][j] = min
		}
	}

	return matrix[len(a)][len(b)]
}

// mapKeys returns the keys of a map[string]value.Value
func mapKeys(m map[string]value.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// builtinIsInt checks whether the argument is an integer value.
// Used as a primitive constraint: 42 |> Int?
func builtinIsInt(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_INT)
}

// builtinIsFloat checks whether the argument is a float value.
// Used as a primitive constraint: 3.14 |> Float?
func builtinIsFloat(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_FLOAT)
}

// builtinIsString checks whether the argument is a string value.
// Used as a primitive constraint: "hello" |> String?
func builtinIsString(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_STRING)
}

// builtinIsBool checks whether the argument is a boolean value.
// Used as a primitive constraint: true |> Bool?
func builtinIsBool(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_BOOL)
}

// builtinIsArray checks whether the argument is an array value.
// Used as a primitive constraint: [1, 2, 3] |> Array?
func builtinIsArray(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_ARRAY)
}

// builtinIsHash checks whether the argument is a hash (map) value.
// Used as a primitive constraint: {a: 1} |> Hash?
func builtinIsHash(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_HASH)
}

// builtinIsTuple checks whether the argument is a tuple value.
// Used as a primitive constraint: (1, 2) |> Tuple?
func builtinIsTuple(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_TUPLE)
}

// builtinIsNull checks whether the argument is null.
// Used as a primitive constraint: null |> Null?
func builtinIsNull(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_NULL)
}

// builtinIsFunction checks whether the argument is a callable function value.
// This includes closures, built-in functions, and overloaded closures.
// Used as a primitive constraint: fn |> Function?
func builtinIsFunction(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	switch args[0].Type {
	case value.TYPE_FUNCTION, value.TYPE_CLOSURE, value.TYPE_BUILTIN,
		value.TYPE_PARTIAL_BUILTIN, value.TYPE_OVERLOADED_CLOSURE:
		return value.Bool(true)
	default:
		return value.Bool(false)
	}
}

// builtinIsNumber checks whether the argument is an integer or float value.
// Used as a primitive constraint: 42 |> Number?
func builtinIsNumber(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Bool(false)
	}
	return value.Bool(args[0].Type == value.TYPE_INT || args[0].Type == value.TYPE_FLOAT)
}
