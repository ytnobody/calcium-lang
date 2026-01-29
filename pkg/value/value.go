package value

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Type represents the type of a Value
type Type int

const (
	TYPE_NULL Type = iota
	TYPE_BOOL
	TYPE_INT
	TYPE_FLOAT
	TYPE_STRING
	TYPE_ARRAY
	TYPE_HASH
	TYPE_FUNCTION
	TYPE_CLOSURE
	TYPE_BUILTIN
	TYPE_PARTIAL_BUILTIN
	TYPE_SUCCESS
	TYPE_FAILURE
	TYPE_MODULE
	TYPE_REGEX        // Compiled regular expression
	TYPE_TASK         // Return value of async.spawn
	TYPE_HANDLER      // Return value of async.expects
	TYPE_EVENT_SOURCE // Event source (stdin, timeout, interval, task.done)
)

func (t Type) String() string {
	switch t {
	case TYPE_NULL:
		return "null"
	case TYPE_BOOL:
		return "bool"
	case TYPE_INT:
		return "int"
	case TYPE_FLOAT:
		return "float"
	case TYPE_STRING:
		return "string"
	case TYPE_ARRAY:
		return "array"
	case TYPE_HASH:
		return "hash"
	case TYPE_FUNCTION:
		return "function"
	case TYPE_CLOSURE:
		return "closure"
	case TYPE_BUILTIN:
		return "builtin"
	case TYPE_PARTIAL_BUILTIN:
		return "partial_builtin"
	case TYPE_SUCCESS:
		return "success"
	case TYPE_FAILURE:
		return "failure"
	case TYPE_MODULE:
		return "module"
	case TYPE_REGEX:
		return "regex"
	case TYPE_TASK:
		return "task"
	case TYPE_HANDLER:
		return "handler"
	case TYPE_EVENT_SOURCE:
		return "event_source"
	default:
		return "unknown"
	}
}

// Value represents a runtime value
type Value struct {
	Type Type
	Data interface{}
}

// Function represents a user-defined function
type Function struct {
	Name       string
	Parameters []string
	Body       []byte     // Bytecode
	NumLocals  int        // Number of local variables
	Constants  []Value    // Constants pool
	Globals    []Value    // Globals pool (for module functions)
	Builtins   []*Builtin // Builtins pool (for module functions)
	IsEffect   bool       // true for func!
}

// BuiltinFn is the signature for built-in functions
type BuiltinFn func(args ...Value) Value

// Builtin represents a built-in function
type Builtin struct {
	Name string
	Fn   BuiltinFn
}

// Module represents a loaded module with exports
type Module struct {
	Name    string           // e.g., "core.io"
	Exports map[string]Value // exported functions/values
}

// Closure represents a function with captured free variables
type Closure struct {
	Fn   *Function // The underlying function
	Free []Value   // Captured free variables
}

// PartialBuiltin represents a partially applied builtin function
type PartialBuiltin struct {
	Name string  // Name of the builtin (map, filter, reduce)
	Args []Value // Already applied arguments
}

// TaskStatus represents the status of an async task
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskCancelled
	TaskFailed
)

// HandlerStatus represents the status of an event handler
type HandlerStatus int

const (
	HandlerDormant HandlerStatus = iota
	HandlerActive
	HandlerPaused
	HandlerCancelled
)

// EventSourceKind represents the kind of event source
type EventSourceKind int

const (
	EventSourceStdin EventSourceKind = iota
	EventSourceTimeout
	EventSourceInterval
	EventSourceTaskDone
	EventSourceEOF
)

// Task represents an async task (result of async.spawn)
type Task struct {
	ID     int64
	mu     sync.RWMutex  // Protects Status, Result, Error
	status TaskStatus
	result Value
	err    error
	Done   *EventSource  // .done property
	Cancel chan struct{} // Cancel signal
}

// SetStatus sets the task status safely
func (t *Task) SetStatus(s TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = s
}

// GetStatus returns the task status safely
func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// SetResult sets the task result safely
func (t *Task) SetResult(v Value) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.result = v
}

// GetResult returns the task result safely
func (t *Task) GetResult() Value {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.result
}

// SetError sets the task error safely
func (t *Task) SetError(e error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = e
}

// GetError returns the task error safely
func (t *Task) GetError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.err
}

// Handler represents an event handler (result of async.expects)
type Handler struct {
	ID       int64
	Status   HandlerStatus
	Source   *EventSource
	Callback Value // Callback function
}

// EventSource represents an event source (stdin, timeout, interval, task.done)
type EventSource struct {
	Kind    EventSourceKind
	Channel chan Value
	Done    chan struct{}
	Closed  bool
}

// Regex represents a compiled regular expression
type Regex struct {
	Pattern string         // Original pattern string
	Flags   string         // Flags (i, m, s)
	Re      *regexp.Regexp // Compiled regular expression
}

// HashPair represents a key-value pair in a hash (preserves insertion order)
type HashPair struct {
	Key   Value
	Value Value
}

// Hash represents a hash map (associative array)
type Hash struct {
	Pairs  []HashPair        // Ordered pairs for iteration
	Lookup map[string]int    // Key (as string) -> index in Pairs for O(1) lookup
}

// Constructors

// Null creates a null value
func Null() Value {
	return Value{Type: TYPE_NULL, Data: nil}
}

// Bool creates a boolean value
func Bool(b bool) Value {
	return Value{Type: TYPE_BOOL, Data: b}
}

// Int creates an integer value
func Int(i int64) Value {
	return Value{Type: TYPE_INT, Data: i}
}

// Float creates a float value
func Float(f float64) Value {
	return Value{Type: TYPE_FLOAT, Data: f}
}

// String creates a string value
func String(s string) Value {
	return Value{Type: TYPE_STRING, Data: s}
}

// Array creates an array value
func Array(elements []Value) Value {
	return Value{Type: TYPE_ARRAY, Data: elements}
}

// Func creates a function value
func Func(fn *Function) Value {
	return Value{Type: TYPE_FUNCTION, Data: fn}
}

// BuiltinFunc creates a builtin function value
func BuiltinFunc(b *Builtin) Value {
	return Value{Type: TYPE_BUILTIN, Data: b}
}

// Success creates a success value (for effect handling)
func Success(v Value) Value {
	return Value{Type: TYPE_SUCCESS, Data: v}
}

// Failure creates a failure value (for effect handling)
func Failure(v Value) Value {
	return Value{Type: TYPE_FAILURE, Data: v}
}

// ModuleVal creates a module value
func ModuleVal(m *Module) Value {
	return Value{Type: TYPE_MODULE, Data: m}
}

// ClosureVal creates a closure value
func ClosureVal(c *Closure) Value {
	return Value{Type: TYPE_CLOSURE, Data: c}
}

// PartialBuiltinVal creates a partial builtin value
func PartialBuiltinVal(p *PartialBuiltin) Value {
	return Value{Type: TYPE_PARTIAL_BUILTIN, Data: p}
}

// HashVal creates a hash value
func HashVal(h *Hash) Value {
	return Value{Type: TYPE_HASH, Data: h}
}

// TaskVal creates a task value
func TaskVal(t *Task) Value {
	return Value{Type: TYPE_TASK, Data: t}
}

// HandlerVal creates a handler value
func HandlerVal(h *Handler) Value {
	return Value{Type: TYPE_HANDLER, Data: h}
}

// EventSourceVal creates an event source value
func EventSourceVal(es *EventSource) Value {
	return Value{Type: TYPE_EVENT_SOURCE, Data: es}
}

// RegexVal creates a regex value
func RegexVal(r *Regex) Value {
	return Value{Type: TYPE_REGEX, Data: r}
}

// NewHash creates a new empty hash
func NewHash() *Hash {
	return &Hash{
		Pairs:  []HashPair{},
		Lookup: make(map[string]int),
	}
}

// Set adds or updates a key-value pair in the hash
func (h *Hash) Set(key, val Value) {
	keyStr := key.String()
	if idx, exists := h.Lookup[keyStr]; exists {
		// Update existing
		h.Pairs[idx].Value = val
	} else {
		// Add new
		h.Lookup[keyStr] = len(h.Pairs)
		h.Pairs = append(h.Pairs, HashPair{Key: key, Value: val})
	}
}

// Get retrieves a value by key, returns (value, true) if found, (null, false) if not
func (h *Hash) Get(key Value) (Value, bool) {
	keyStr := key.String()
	if idx, exists := h.Lookup[keyStr]; exists {
		return h.Pairs[idx].Value, true
	}
	return Null(), false
}

// Keys returns all keys in insertion order
func (h *Hash) Keys() []Value {
	keys := make([]Value, len(h.Pairs))
	for i, pair := range h.Pairs {
		keys[i] = pair.Key
	}
	return keys
}

// Values returns all values in insertion order
func (h *Hash) Values() []Value {
	values := make([]Value, len(h.Pairs))
	for i, pair := range h.Pairs {
		values[i] = pair.Value
	}
	return values
}

// Accessors

// AsBool returns the boolean value
func (v Value) AsBool() bool {
	return v.Data.(bool)
}

// AsInt returns the integer value
func (v Value) AsInt() int64 {
	return v.Data.(int64)
}

// AsFloat returns the float value
func (v Value) AsFloat() float64 {
	return v.Data.(float64)
}

// AsString returns the string value
func (v Value) AsString() string {
	return v.Data.(string)
}

// AsArray returns the array value
func (v Value) AsArray() []Value {
	return v.Data.([]Value)
}

// AsFunction returns the function value
func (v Value) AsFunction() *Function {
	return v.Data.(*Function)
}

// AsBuiltin returns the builtin function
func (v Value) AsBuiltin() *Builtin {
	return v.Data.(*Builtin)
}

// AsSuccess returns the unwrapped success value
func (v Value) AsSuccess() Value {
	return v.Data.(Value)
}

// AsFailure returns the unwrapped failure value
func (v Value) AsFailure() Value {
	return v.Data.(Value)
}

// AsModule returns the module value
func (v Value) AsModule() *Module {
	return v.Data.(*Module)
}

// AsClosure returns the closure value
func (v Value) AsClosure() *Closure {
	return v.Data.(*Closure)
}

// AsPartialBuiltin returns the partial builtin value
func (v Value) AsPartialBuiltin() *PartialBuiltin {
	return v.Data.(*PartialBuiltin)
}

// AsHash returns the hash value
func (v Value) AsHash() *Hash {
	return v.Data.(*Hash)
}

// AsTask returns the task value
func (v Value) AsTask() *Task {
	return v.Data.(*Task)
}

// AsHandler returns the handler value
func (v Value) AsHandler() *Handler {
	return v.Data.(*Handler)
}

// AsEventSource returns the event source value
func (v Value) AsEventSource() *EventSource {
	return v.Data.(*EventSource)
}

// AsRegex returns the regex value
func (v Value) AsRegex() *Regex {
	return v.Data.(*Regex)
}

// ToNumber converts value to a numeric type for arithmetic
func (v Value) ToNumber() (float64, bool) {
	switch v.Type {
	case TYPE_INT:
		return float64(v.AsInt()), true
	case TYPE_FLOAT:
		return v.AsFloat(), true
	default:
		return 0, false
	}
}

// IsTruthy returns whether the value is truthy
func (v Value) IsTruthy() bool {
	switch v.Type {
	case TYPE_NULL:
		return false
	case TYPE_BOOL:
		return v.AsBool()
	case TYPE_INT:
		return v.AsInt() != 0
	case TYPE_FLOAT:
		return v.AsFloat() != 0
	case TYPE_STRING:
		return v.AsString() != ""
	case TYPE_ARRAY:
		return len(v.AsArray()) > 0
	case TYPE_SUCCESS:
		return true
	case TYPE_FAILURE:
		return false
	default:
		return true
	}
}

// IsSuccess checks if value is a success
func (v Value) IsSuccess() bool {
	return v.Type == TYPE_SUCCESS
}

// IsFailure checks if value is a failure
func (v Value) IsFailure() bool {
	return v.Type == TYPE_FAILURE
}

// Equals checks value equality
func (v Value) Equals(other Value) bool {
	if v.Type != other.Type {
		// Special case: int and float comparison
		if (v.Type == TYPE_INT || v.Type == TYPE_FLOAT) &&
			(other.Type == TYPE_INT || other.Type == TYPE_FLOAT) {
			vn, _ := v.ToNumber()
			on, _ := other.ToNumber()
			return vn == on
		}
		return false
	}

	switch v.Type {
	case TYPE_NULL:
		return true
	case TYPE_BOOL:
		return v.AsBool() == other.AsBool()
	case TYPE_INT:
		return v.AsInt() == other.AsInt()
	case TYPE_FLOAT:
		return v.AsFloat() == other.AsFloat()
	case TYPE_STRING:
		return v.AsString() == other.AsString()
	case TYPE_ARRAY:
		a1, a2 := v.AsArray(), other.AsArray()
		if len(a1) != len(a2) {
			return false
		}
		for i := range a1 {
			if !a1[i].Equals(a2[i]) {
				return false
			}
		}
		return true
	case TYPE_HASH:
		h1, h2 := v.AsHash(), other.AsHash()
		if len(h1.Pairs) != len(h2.Pairs) {
			return false
		}
		for _, pair := range h1.Pairs {
			val, exists := h2.Get(pair.Key)
			if !exists || !pair.Value.Equals(val) {
				return false
			}
		}
		return true
	case TYPE_SUCCESS:
		return v.AsSuccess().Equals(other.AsSuccess())
	case TYPE_FAILURE:
		return v.AsFailure().Equals(other.AsFailure())
	default:
		return false
	}
}

// String returns a string representation of the value
func (v Value) String() string {
	switch v.Type {
	case TYPE_NULL:
		return "null"
	case TYPE_BOOL:
		if v.AsBool() {
			return "true"
		}
		return "false"
	case TYPE_INT:
		return fmt.Sprintf("%d", v.AsInt())
	case TYPE_FLOAT:
		return fmt.Sprintf("%g", v.AsFloat())
	case TYPE_STRING:
		return v.AsString()
	case TYPE_ARRAY:
		elements := v.AsArray()
		strs := make([]string, len(elements))
		for i, e := range elements {
			strs[i] = e.String()
		}
		return "[" + strings.Join(strs, ", ") + "]"
	case TYPE_HASH:
		h := v.AsHash()
		strs := make([]string, len(h.Pairs))
		for i, pair := range h.Pairs {
			strs[i] = pair.Key.String() + ": " + pair.Value.String()
		}
		return "{" + strings.Join(strs, ", ") + "}"
	case TYPE_FUNCTION:
		fn := v.AsFunction()
		if fn.IsEffect {
			return fmt.Sprintf("<func! %s>", fn.Name)
		}
		return fmt.Sprintf("<func %s>", fn.Name)
	case TYPE_CLOSURE:
		cl := v.AsClosure()
		if cl.Fn.IsEffect {
			return fmt.Sprintf("<func! %s>", cl.Fn.Name)
		}
		return fmt.Sprintf("<func %s>", cl.Fn.Name)
	case TYPE_BUILTIN:
		return fmt.Sprintf("<builtin %s>", v.AsBuiltin().Name)
	case TYPE_PARTIAL_BUILTIN:
		return fmt.Sprintf("<partial %s>", v.AsPartialBuiltin().Name)
	case TYPE_SUCCESS:
		return fmt.Sprintf("success(%s)", v.AsSuccess().String())
	case TYPE_FAILURE:
		return fmt.Sprintf("failure(%s)", v.AsFailure().String())
	case TYPE_MODULE:
		return fmt.Sprintf("<module %s>", v.AsModule().Name)
	case TYPE_REGEX:
		r := v.AsRegex()
		return "/" + r.Pattern + "/" + r.Flags
	case TYPE_TASK:
		t := v.AsTask()
		statusStr := "pending"
		switch t.GetStatus() {
		case TaskRunning:
			statusStr = "running"
		case TaskCompleted:
			statusStr = "completed"
		case TaskCancelled:
			statusStr = "cancelled"
		case TaskFailed:
			statusStr = "failed"
		}
		return fmt.Sprintf("<task %d:%s>", t.ID, statusStr)
	case TYPE_HANDLER:
		h := v.AsHandler()
		statusStr := "dormant"
		switch h.Status {
		case HandlerActive:
			statusStr = "active"
		case HandlerPaused:
			statusStr = "paused"
		case HandlerCancelled:
			statusStr = "cancelled"
		}
		return fmt.Sprintf("<handler %d:%s>", h.ID, statusStr)
	case TYPE_EVENT_SOURCE:
		es := v.AsEventSource()
		kindStr := "unknown"
		switch es.Kind {
		case EventSourceStdin:
			kindStr = "stdin"
		case EventSourceTimeout:
			kindStr = "timeout"
		case EventSourceInterval:
			kindStr = "interval"
		case EventSourceTaskDone:
			kindStr = "task.done"
		case EventSourceEOF:
			kindStr = "eof"
		}
		return fmt.Sprintf("<event_source:%s>", kindStr)
	default:
		return "<unknown>"
	}
}
