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
	TYPE_TUPLE        // Tuple (fixed-length, heterogeneous)
	TYPE_REGEX        // Compiled regular expression
	TYPE_TASK         // Return value of async.spawn
	TYPE_HANDLER      // Return value of async.expects
	TYPE_EVENT_SOURCE // Event source (stdin, timeout, interval, task.done)
	TYPE_ADT                // Algebraic data type variant value
	TYPE_CHANNEL            // User-level message passing channel
	TYPE_OVERLOADED_CLOSURE // Multiple function overloads with different signatures
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
	case TYPE_TUPLE:
		return "tuple"
	case TYPE_REGEX:
		return "regex"
	case TYPE_TASK:
		return "task"
	case TYPE_HANDLER:
		return "handler"
	case TYPE_EVENT_SOURCE:
		return "event_source"
	case TYPE_ADT:
		return "adt"
	case TYPE_CHANNEL:
		return "channel"
	case TYPE_OVERLOADED_CLOSURE:
		return "overloaded_closure"
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
	Name           string
	Parameters     []string
	ParamTypeNames []string    // type annotation names for each parameter (empty string = untyped/any)
	Body           []byte      // Bytecode
	NumLocals      int         // Number of local variables
	Constants      []Value     // Constants pool
	Globals        []Value     // Globals pool (for module functions)
	Builtins       []*Builtin  // Builtins pool (for module functions)
	IsEffect       bool        // true for func!
	SourceMap      interface{} // *bytecode.SourceMap (stored as interface to avoid circular import)
}

// OverloadedClosure holds multiple function variants with the same name but different signatures.
// At call time, the appropriate variant is selected based on argument count and types.
type OverloadedClosure struct {
	Name     string      // function name (shared by all variants)
	Variants []*Closure  // variants ordered by definition order
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
	EventSourceChannel // Event source backed by a user-level channel
)

// Task represents an async task (result of async.spawn)
type Task struct {
	ID     int64
	mu     sync.RWMutex // Protects Status, Result, Error
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

// ADT represents an algebraic data type variant value
type ADT struct {
	TypeName string  // e.g., "Maybe", "Tree"
	Tag      string  // e.g., "Some", "None"
	Values   []Value // contained values
}

// Channel represents a user-level message passing channel (created by async.channel)
//
// Design: The internal EventSource.Channel IS the shared buffer. Both ch.receive()
// and the event handler goroutine (when ch.source is used with async.expects) compete
// for values. This provides "consume once" semantics consistent with channel-based
// message passing.
type Channel struct {
	mu       sync.Mutex
	closed   bool
	capacity int
	Source   *EventSource // EventSource for use with async.expects; its Channel is the shared buffer
}

// NewChannel creates a new channel with the given capacity (0 = unbuffered).
// The channel's internal buffer is backed by the EventSource.Channel.
func NewChannel(capacity int) *Channel {
	bufSize := capacity
	if bufSize <= 0 {
		// Unbuffered-style: allow a small buffer to avoid deadlock in stay loops
		// when the event dispatcher reads values from the source channel.
		bufSize = 1
	}
	es := &EventSource{
		Kind: EventSourceChannel,
		// Use max(bufSize, 64) so the event dispatcher doesn't block delivery.
		Channel: make(chan Value, max(bufSize, 64)),
		Done:    make(chan struct{}),
	}
	return &Channel{
		capacity: capacity,
		Source:   es,
	}
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Send sends a value to the channel. Returns false if the channel is closed.
func (c *Channel) Send(val Value) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()

	select {
	case c.Source.Channel <- val:
		return true
	case <-c.Source.Done:
		return false
	}
}

// Receive receives a value from the channel. Blocks until a value is available
// or the channel is closed.
func (c *Channel) Receive() (Value, bool) {
	select {
	case val, ok := <-c.Source.Channel:
		if !ok {
			return Null(), false
		}
		return val, true
	case <-c.Source.Done:
		return Null(), false
	}
}

// Close closes the channel.
// Close closes the channel, preventing further sends and signaling receivers.
func (c *Channel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		// Signal the Done channel to unblock any waiting receivers/handlers
		select {
		case <-c.Source.Done:
			// Already closed
		default:
			close(c.Source.Done)
		}
	}
}

// IsClosed returns true if the channel is closed.
func (c *Channel) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// ADTDef represents the definition of an algebraic data type
type ADTDef struct {
	Name     string         // e.g., "Maybe"
	Variants map[string]int // variant name -> arity
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
	Pairs  []HashPair     // Ordered pairs for iteration
	Lookup map[string]int // Key (as string) -> index in Pairs for O(1) lookup
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

// Tuple creates a tuple value
func Tuple(elements []Value) Value {
	return Value{Type: TYPE_TUPLE, Data: elements}
}

// RegexVal creates a regex value
func RegexVal(r *Regex) Value {
	return Value{Type: TYPE_REGEX, Data: r}
}

// ADTVal creates an ADT variant value
func ADTVal(adt *ADT) Value {
	return Value{Type: TYPE_ADT, Data: adt}
}

// ChannelVal creates a channel value
func ChannelVal(c *Channel) Value {
	return Value{Type: TYPE_CHANNEL, Data: c}
}

// OverloadedClosureVal creates an overloaded closure value
func OverloadedClosureVal(oc *OverloadedClosure) Value {
	return Value{Type: TYPE_OVERLOADED_CLOSURE, Data: oc}
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

// AsTuple returns the tuple elements
func (v Value) AsTuple() []Value {
	return v.Data.([]Value)
}

// AsRegex returns the regex value
func (v Value) AsRegex() *Regex {
	return v.Data.(*Regex)
}

// AsADT returns the ADT variant value
func (v Value) AsADT() *ADT {
	return v.Data.(*ADT)
}

// AsChannel returns the channel value
func (v Value) AsChannel() *Channel {
	return v.Data.(*Channel)
}

// AsOverloadedClosure returns the overloaded closure value
func (v Value) AsOverloadedClosure() *OverloadedClosure {
	return v.Data.(*OverloadedClosure)
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
	case TYPE_TUPLE:
		return len(v.AsTuple()) > 0
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
	case TYPE_TUPLE:
		t1, t2 := v.AsTuple(), other.AsTuple()
		if len(t1) != len(t2) {
			return false
		}
		for i := range t1 {
			if !t1[i].Equals(t2[i]) {
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
	case TYPE_ADT:
		a1, a2 := v.AsADT(), other.AsADT()
		if a1.Tag != a2.Tag || a1.TypeName != a2.TypeName || len(a1.Values) != len(a2.Values) {
			return false
		}
		for i := range a1.Values {
			if !a1.Values[i].Equals(a2.Values[i]) {
				return false
			}
		}
		return true
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
	case TYPE_TUPLE:
		elements := v.AsTuple()
		strs := make([]string, len(elements))
		for i, e := range elements {
			strs[i] = e.String()
		}
		return "(" + strings.Join(strs, ", ") + ")"
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
	case TYPE_ADT:
		adt := v.AsADT()
		if len(adt.Values) == 0 {
			return adt.Tag
		}
		strs := make([]string, len(adt.Values))
		for i, val := range adt.Values {
			strs[i] = val.String()
		}
		return fmt.Sprintf("%s(%s)", adt.Tag, strings.Join(strs, ", "))
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
		case EventSourceChannel:
			kindStr = "channel"
		}
		return fmt.Sprintf("<event_source:%s>", kindStr)
	case TYPE_CHANNEL:
		c := v.AsChannel()
		if c.IsClosed() {
			return "<channel:closed>"
		}
		return fmt.Sprintf("<channel:open(cap=%d)>", c.capacity)
	case TYPE_OVERLOADED_CLOSURE:
		oc := v.AsOverloadedClosure()
		return fmt.Sprintf("<overloaded func %s (%d variants)>", oc.Name, len(oc.Variants))
	default:
		return "<unknown>"
	}
}
