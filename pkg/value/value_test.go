package value

import (
	"regexp"
	"sync"
	"testing"
)

// Test Type.String()
func TestTypeString(t *testing.T) {
	tests := []struct {
		typ  Type
		want string
	}{
		{TYPE_NULL, "null"},
		{TYPE_BOOL, "bool"},
		{TYPE_INT, "int"},
		{TYPE_FLOAT, "float"},
		{TYPE_STRING, "string"},
		{TYPE_ARRAY, "array"},
		{TYPE_HASH, "hash"},
		{TYPE_FUNCTION, "function"},
		{TYPE_CLOSURE, "closure"},
		{TYPE_BUILTIN, "builtin"},
		{TYPE_PARTIAL_BUILTIN, "partial_builtin"},
		{TYPE_SUCCESS, "success"},
		{TYPE_FAILURE, "failure"},
		{TYPE_MODULE, "module"},
		{TYPE_REGEX, "regex"},
		{TYPE_TASK, "task"},
		{TYPE_HANDLER, "handler"},
		{TYPE_EVENT_SOURCE, "event_source"},
		{Type(999), "unknown"},
	}

	for _, tt := range tests {
		got := tt.typ.String()
		if got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

// Test constructors
func TestConstructors(t *testing.T) {
	t.Run("Null", func(t *testing.T) {
		v := Null()
		if v.Type != TYPE_NULL {
			t.Errorf("Null().Type = %v, want TYPE_NULL", v.Type)
		}
		if v.Data != nil {
			t.Errorf("Null().Data = %v, want nil", v.Data)
		}
	})

	t.Run("Bool", func(t *testing.T) {
		v := Bool(true)
		if v.Type != TYPE_BOOL {
			t.Errorf("Bool(true).Type = %v, want TYPE_BOOL", v.Type)
		}
		if !v.AsBool() {
			t.Error("Bool(true).AsBool() = false, want true")
		}

		v = Bool(false)
		if v.AsBool() {
			t.Error("Bool(false).AsBool() = true, want false")
		}
	})

	t.Run("Int", func(t *testing.T) {
		v := Int(42)
		if v.Type != TYPE_INT {
			t.Errorf("Int(42).Type = %v, want TYPE_INT", v.Type)
		}
		if v.AsInt() != 42 {
			t.Errorf("Int(42).AsInt() = %d, want 42", v.AsInt())
		}
	})

	t.Run("Float", func(t *testing.T) {
		v := Float(3.14)
		if v.Type != TYPE_FLOAT {
			t.Errorf("Float(3.14).Type = %v, want TYPE_FLOAT", v.Type)
		}
		if v.AsFloat() != 3.14 {
			t.Errorf("Float(3.14).AsFloat() = %f, want 3.14", v.AsFloat())
		}
	})

	t.Run("String", func(t *testing.T) {
		v := String("hello")
		if v.Type != TYPE_STRING {
			t.Errorf("String(hello).Type = %v, want TYPE_STRING", v.Type)
		}
		if v.AsString() != "hello" {
			t.Errorf("String(hello).AsString() = %q, want hello", v.AsString())
		}
	})

	t.Run("Array", func(t *testing.T) {
		elements := []Value{Int(1), Int(2), Int(3)}
		v := Array(elements)
		if v.Type != TYPE_ARRAY {
			t.Errorf("Array().Type = %v, want TYPE_ARRAY", v.Type)
		}
		arr := v.AsArray()
		if len(arr) != 3 {
			t.Errorf("len(Array().AsArray()) = %d, want 3", len(arr))
		}
	})

	t.Run("Func", func(t *testing.T) {
		fn := &Function{Name: "test", Parameters: []string{"a"}}
		v := Func(fn)
		if v.Type != TYPE_FUNCTION {
			t.Errorf("Func().Type = %v, want TYPE_FUNCTION", v.Type)
		}
		if v.AsFunction().Name != "test" {
			t.Errorf("Func().AsFunction().Name = %q, want test", v.AsFunction().Name)
		}
	})

	t.Run("BuiltinFunc", func(t *testing.T) {
		b := &Builtin{Name: "print", Fn: func(args ...Value) Value { return Null() }}
		v := BuiltinFunc(b)
		if v.Type != TYPE_BUILTIN {
			t.Errorf("BuiltinFunc().Type = %v, want TYPE_BUILTIN", v.Type)
		}
		if v.AsBuiltin().Name != "print" {
			t.Errorf("BuiltinFunc().AsBuiltin().Name = %q, want print", v.AsBuiltin().Name)
		}
	})

	t.Run("Success", func(t *testing.T) {
		inner := Int(42)
		v := Success(inner)
		if v.Type != TYPE_SUCCESS {
			t.Errorf("Success().Type = %v, want TYPE_SUCCESS", v.Type)
		}
		if !v.AsSuccess().Equals(inner) {
			t.Error("Success(Int(42)).AsSuccess() != Int(42)")
		}
	})

	t.Run("Failure", func(t *testing.T) {
		inner := String("error")
		v := Failure(inner)
		if v.Type != TYPE_FAILURE {
			t.Errorf("Failure().Type = %v, want TYPE_FAILURE", v.Type)
		}
		if !v.AsFailure().Equals(inner) {
			t.Error("Failure(String(error)).AsFailure() != String(error)")
		}
	})

	t.Run("ModuleVal", func(t *testing.T) {
		m := &Module{Name: "core.io", Exports: make(map[string]Value)}
		v := ModuleVal(m)
		if v.Type != TYPE_MODULE {
			t.Errorf("ModuleVal().Type = %v, want TYPE_MODULE", v.Type)
		}
		if v.AsModule().Name != "core.io" {
			t.Errorf("ModuleVal().AsModule().Name = %q, want core.io", v.AsModule().Name)
		}
	})

	t.Run("ClosureVal", func(t *testing.T) {
		fn := &Function{Name: "closure_fn"}
		c := &Closure{Fn: fn, Free: []Value{Int(1)}}
		v := ClosureVal(c)
		if v.Type != TYPE_CLOSURE {
			t.Errorf("ClosureVal().Type = %v, want TYPE_CLOSURE", v.Type)
		}
		if v.AsClosure().Fn.Name != "closure_fn" {
			t.Errorf("ClosureVal().AsClosure().Fn.Name = %q, want closure_fn", v.AsClosure().Fn.Name)
		}
	})

	t.Run("PartialBuiltinVal", func(t *testing.T) {
		p := &PartialBuiltin{Name: "map", Args: []Value{Int(1)}}
		v := PartialBuiltinVal(p)
		if v.Type != TYPE_PARTIAL_BUILTIN {
			t.Errorf("PartialBuiltinVal().Type = %v, want TYPE_PARTIAL_BUILTIN", v.Type)
		}
		if v.AsPartialBuiltin().Name != "map" {
			t.Errorf("PartialBuiltinVal().AsPartialBuiltin().Name = %q, want map", v.AsPartialBuiltin().Name)
		}
	})

	t.Run("HashVal", func(t *testing.T) {
		h := NewHash()
		h.Set(String("key"), Int(42))
		v := HashVal(h)
		if v.Type != TYPE_HASH {
			t.Errorf("HashVal().Type = %v, want TYPE_HASH", v.Type)
		}
		if len(v.AsHash().Pairs) != 1 {
			t.Errorf("len(HashVal().AsHash().Pairs) = %d, want 1", len(v.AsHash().Pairs))
		}
	})

	t.Run("TaskVal", func(t *testing.T) {
		task := &Task{ID: 123}
		v := TaskVal(task)
		if v.Type != TYPE_TASK {
			t.Errorf("TaskVal().Type = %v, want TYPE_TASK", v.Type)
		}
		if v.AsTask().ID != 123 {
			t.Errorf("TaskVal().AsTask().ID = %d, want 123", v.AsTask().ID)
		}
	})

	t.Run("HandlerVal", func(t *testing.T) {
		h := &Handler{ID: 456, Status: HandlerActive}
		v := HandlerVal(h)
		if v.Type != TYPE_HANDLER {
			t.Errorf("HandlerVal().Type = %v, want TYPE_HANDLER", v.Type)
		}
		if v.AsHandler().ID != 456 {
			t.Errorf("HandlerVal().AsHandler().ID = %d, want 456", v.AsHandler().ID)
		}
	})

	t.Run("EventSourceVal", func(t *testing.T) {
		es := &EventSource{Kind: EventSourceStdin}
		v := EventSourceVal(es)
		if v.Type != TYPE_EVENT_SOURCE {
			t.Errorf("EventSourceVal().Type = %v, want TYPE_EVENT_SOURCE", v.Type)
		}
		if v.AsEventSource().Kind != EventSourceStdin {
			t.Errorf("EventSourceVal().AsEventSource().Kind = %v, want EventSourceStdin", v.AsEventSource().Kind)
		}
	})

	t.Run("RegexVal", func(t *testing.T) {
		re := regexp.MustCompile("^test$")
		r := &Regex{Pattern: "^test$", Flags: "i", Re: re}
		v := RegexVal(r)
		if v.Type != TYPE_REGEX {
			t.Errorf("RegexVal().Type = %v, want TYPE_REGEX", v.Type)
		}
		if v.AsRegex().Pattern != "^test$" {
			t.Errorf("RegexVal().AsRegex().Pattern = %q, want ^test$", v.AsRegex().Pattern)
		}
	})
}

// Test Hash operations
func TestHash(t *testing.T) {
	t.Run("NewHash", func(t *testing.T) {
		h := NewHash()
		if h == nil {
			t.Fatal("NewHash() returned nil")
		}
		if len(h.Pairs) != 0 {
			t.Errorf("NewHash().Pairs length = %d, want 0", len(h.Pairs))
		}
		if h.Lookup == nil {
			t.Error("NewHash().Lookup is nil")
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		h := NewHash()
		h.Set(String("name"), String("Alice"))
		h.Set(String("age"), Int(30))

		val, exists := h.Get(String("name"))
		if !exists {
			t.Error("Get(name) returned exists=false")
		}
		if val.AsString() != "Alice" {
			t.Errorf("Get(name) = %v, want Alice", val.AsString())
		}

		val, exists = h.Get(String("age"))
		if !exists {
			t.Error("Get(age) returned exists=false")
		}
		if val.AsInt() != 30 {
			t.Errorf("Get(age) = %v, want 30", val.AsInt())
		}

		_, exists = h.Get(String("nonexistent"))
		if exists {
			t.Error("Get(nonexistent) returned exists=true")
		}
	})

	t.Run("Set update", func(t *testing.T) {
		h := NewHash()
		h.Set(String("key"), Int(1))
		h.Set(String("key"), Int(2))

		if len(h.Pairs) != 1 {
			t.Errorf("after update, Pairs length = %d, want 1", len(h.Pairs))
		}
		val, _ := h.Get(String("key"))
		if val.AsInt() != 2 {
			t.Errorf("after update, Get(key) = %d, want 2", val.AsInt())
		}
	})

	t.Run("Keys", func(t *testing.T) {
		h := NewHash()
		h.Set(String("a"), Int(1))
		h.Set(String("b"), Int(2))
		h.Set(String("c"), Int(3))

		keys := h.Keys()
		if len(keys) != 3 {
			t.Errorf("Keys() length = %d, want 3", len(keys))
		}
		// Order should be preserved
		if keys[0].AsString() != "a" || keys[1].AsString() != "b" || keys[2].AsString() != "c" {
			t.Errorf("Keys() order incorrect: %v", keys)
		}
	})

	t.Run("Values", func(t *testing.T) {
		h := NewHash()
		h.Set(String("a"), Int(1))
		h.Set(String("b"), Int(2))
		h.Set(String("c"), Int(3))

		values := h.Values()
		if len(values) != 3 {
			t.Errorf("Values() length = %d, want 3", len(values))
		}
		// Order should be preserved
		if values[0].AsInt() != 1 || values[1].AsInt() != 2 || values[2].AsInt() != 3 {
			t.Errorf("Values() order incorrect: %v", values)
		}
	})
}

// Test ToNumber
func TestToNumber(t *testing.T) {
	tests := []struct {
		val    Value
		want   float64
		wantOk bool
	}{
		{Int(42), 42.0, true},
		{Int(-10), -10.0, true},
		{Float(3.14), 3.14, true},
		{String("hello"), 0, false},
		{Null(), 0, false},
		{Bool(true), 0, false},
	}

	for _, tt := range tests {
		got, ok := tt.val.ToNumber()
		if ok != tt.wantOk {
			t.Errorf("%v.ToNumber() ok = %v, want %v", tt.val, ok, tt.wantOk)
		}
		if ok && got != tt.want {
			t.Errorf("%v.ToNumber() = %v, want %v", tt.val, got, tt.want)
		}
	}
}

// Test IsTruthy
func TestIsTruthy(t *testing.T) {
	tests := []struct {
		val  Value
		want bool
	}{
		{Null(), false},
		{Bool(true), true},
		{Bool(false), false},
		{Int(0), false},
		{Int(1), true},
		{Int(-1), true},
		{Float(0.0), false},
		{Float(0.1), true},
		{String(""), false},
		{String("hello"), true},
		{Array([]Value{}), false},
		{Array([]Value{Int(1)}), true},
		{Success(Int(1)), true},
		{Failure(String("err")), false},
		{Func(&Function{Name: "f"}), true},
	}

	for _, tt := range tests {
		got := tt.val.IsTruthy()
		if got != tt.want {
			t.Errorf("%v.IsTruthy() = %v, want %v", tt.val, got, tt.want)
		}
	}
}

// Test IsSuccess and IsFailure
func TestIsSuccessFailure(t *testing.T) {
	s := Success(Int(1))
	f := Failure(String("error"))
	i := Int(42)

	if !s.IsSuccess() {
		t.Error("Success().IsSuccess() = false")
	}
	if s.IsFailure() {
		t.Error("Success().IsFailure() = true")
	}
	if f.IsSuccess() {
		t.Error("Failure().IsSuccess() = true")
	}
	if !f.IsFailure() {
		t.Error("Failure().IsFailure() = false")
	}
	if i.IsSuccess() || i.IsFailure() {
		t.Error("Int().IsSuccess() or IsFailure() = true")
	}
}

// Test Equals
func TestEquals(t *testing.T) {
	t.Run("same type", func(t *testing.T) {
		tests := []struct {
			a, b Value
			want bool
		}{
			{Null(), Null(), true},
			{Bool(true), Bool(true), true},
			{Bool(true), Bool(false), false},
			{Int(42), Int(42), true},
			{Int(42), Int(43), false},
			{Float(3.14), Float(3.14), true},
			{Float(3.14), Float(3.15), false},
			{String("hello"), String("hello"), true},
			{String("hello"), String("world"), false},
			{Array([]Value{Int(1), Int(2)}), Array([]Value{Int(1), Int(2)}), true},
			{Array([]Value{Int(1), Int(2)}), Array([]Value{Int(1), Int(3)}), false},
			{Array([]Value{Int(1)}), Array([]Value{Int(1), Int(2)}), false},
			{Success(Int(1)), Success(Int(1)), true},
			{Success(Int(1)), Success(Int(2)), false},
			{Failure(String("a")), Failure(String("a")), true},
			{Failure(String("a")), Failure(String("b")), false},
		}

		for _, tt := range tests {
			got := tt.a.Equals(tt.b)
			if got != tt.want {
				t.Errorf("%v.Equals(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("int and float comparison", func(t *testing.T) {
		if !Int(42).Equals(Float(42.0)) {
			t.Error("Int(42).Equals(Float(42.0)) = false")
		}
		if !Float(42.0).Equals(Int(42)) {
			t.Error("Float(42.0).Equals(Int(42)) = false")
		}
		if Int(42).Equals(Float(42.5)) {
			t.Error("Int(42).Equals(Float(42.5)) = true")
		}
	})

	t.Run("different types", func(t *testing.T) {
		if Int(1).Equals(String("1")) {
			t.Error("Int(1).Equals(String(1)) = true")
		}
		if Bool(true).Equals(Int(1)) {
			t.Error("Bool(true).Equals(Int(1)) = true")
		}
	})

	t.Run("hash equality", func(t *testing.T) {
		h1 := NewHash()
		h1.Set(String("a"), Int(1))
		h1.Set(String("b"), Int(2))

		h2 := NewHash()
		h2.Set(String("a"), Int(1))
		h2.Set(String("b"), Int(2))

		h3 := NewHash()
		h3.Set(String("a"), Int(1))
		h3.Set(String("b"), Int(3))

		if !HashVal(h1).Equals(HashVal(h2)) {
			t.Error("equal hashes not equal")
		}
		if HashVal(h1).Equals(HashVal(h3)) {
			t.Error("different hashes are equal")
		}
	})

	t.Run("unsupported type returns false", func(t *testing.T) {
		fn1 := Func(&Function{Name: "f"})
		fn2 := Func(&Function{Name: "f"})
		if fn1.Equals(fn2) {
			t.Error("function Equals should return false")
		}
	})
}

// Test Value.String()
func TestValueString(t *testing.T) {
	tests := []struct {
		val  Value
		want string
	}{
		{Null(), "null"},
		{Bool(true), "true"},
		{Bool(false), "false"},
		{Int(42), "42"},
		{Int(-123), "-123"},
		{Float(3.14), "3.14"},
		{Float(1e10), "1e+10"},
		{String("hello"), "hello"},
		{Array([]Value{}), "[]"},
		{Array([]Value{Int(1), Int(2)}), "[1, 2]"},
		{Success(Int(42)), "success(42)"},
		{Failure(String("error")), "failure(error)"},
	}

	for _, tt := range tests {
		got := tt.val.String()
		if got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.val.Type, got, tt.want)
		}
	}
}

func TestValueStringComplex(t *testing.T) {
	t.Run("Hash", func(t *testing.T) {
		h := NewHash()
		h.Set(String("a"), Int(1))
		v := HashVal(h)
		got := v.String()
		if got != "{a: 1}" {
			t.Errorf("Hash.String() = %q, want {a: 1}", got)
		}
	})

	t.Run("Function", func(t *testing.T) {
		fn := &Function{Name: "myFunc", IsEffect: false}
		v := Func(fn)
		if v.String() != "<func myFunc>" {
			t.Errorf("Function.String() = %q, want <func myFunc>", v.String())
		}

		fn.IsEffect = true
		if v.String() != "<func! myFunc>" {
			t.Errorf("EffectFunction.String() = %q, want <func! myFunc>", v.String())
		}
	})

	t.Run("Closure", func(t *testing.T) {
		fn := &Function{Name: "closureFn", IsEffect: false}
		c := &Closure{Fn: fn, Free: []Value{}}
		v := ClosureVal(c)
		if v.String() != "<func closureFn>" {
			t.Errorf("Closure.String() = %q, want <func closureFn>", v.String())
		}

		fn.IsEffect = true
		if v.String() != "<func! closureFn>" {
			t.Errorf("EffectClosure.String() = %q, want <func! closureFn>", v.String())
		}
	})

	t.Run("Builtin", func(t *testing.T) {
		b := &Builtin{Name: "print"}
		v := BuiltinFunc(b)
		if v.String() != "<builtin print>" {
			t.Errorf("Builtin.String() = %q, want <builtin print>", v.String())
		}
	})

	t.Run("PartialBuiltin", func(t *testing.T) {
		p := &PartialBuiltin{Name: "map"}
		v := PartialBuiltinVal(p)
		if v.String() != "<partial map>" {
			t.Errorf("PartialBuiltin.String() = %q, want <partial map>", v.String())
		}
	})

	t.Run("Module", func(t *testing.T) {
		m := &Module{Name: "core.io"}
		v := ModuleVal(m)
		if v.String() != "<module core.io>" {
			t.Errorf("Module.String() = %q, want <module core.io>", v.String())
		}
	})

	t.Run("Regex", func(t *testing.T) {
		r := &Regex{Pattern: "^test$", Flags: "im"}
		v := RegexVal(r)
		if v.String() != "/^test$/im" {
			t.Errorf("Regex.String() = %q, want /^test$/im", v.String())
		}
	})

	t.Run("Task", func(t *testing.T) {
		task := &Task{ID: 1}
		v := TaskVal(task)

		task.SetStatus(TaskPending)
		if v.String() != "<task 1:pending>" {
			t.Errorf("Task(pending).String() = %q", v.String())
		}

		task.SetStatus(TaskRunning)
		if v.String() != "<task 1:running>" {
			t.Errorf("Task(running).String() = %q", v.String())
		}

		task.SetStatus(TaskCompleted)
		if v.String() != "<task 1:completed>" {
			t.Errorf("Task(completed).String() = %q", v.String())
		}

		task.SetStatus(TaskCancelled)
		if v.String() != "<task 1:cancelled>" {
			t.Errorf("Task(cancelled).String() = %q", v.String())
		}

		task.SetStatus(TaskFailed)
		if v.String() != "<task 1:failed>" {
			t.Errorf("Task(failed).String() = %q", v.String())
		}
	})

	t.Run("Handler", func(t *testing.T) {
		h := &Handler{ID: 2, Status: HandlerDormant}
		v := HandlerVal(h)
		if v.String() != "<handler 2:dormant>" {
			t.Errorf("Handler(dormant).String() = %q", v.String())
		}

		h.Status = HandlerActive
		if v.String() != "<handler 2:active>" {
			t.Errorf("Handler(active).String() = %q", v.String())
		}

		h.Status = HandlerPaused
		if v.String() != "<handler 2:paused>" {
			t.Errorf("Handler(paused).String() = %q", v.String())
		}

		h.Status = HandlerCancelled
		if v.String() != "<handler 2:cancelled>" {
			t.Errorf("Handler(cancelled).String() = %q", v.String())
		}
	})

	t.Run("EventSource", func(t *testing.T) {
		tests := []struct {
			kind EventSourceKind
			want string
		}{
			{EventSourceStdin, "<event_source:stdin>"},
			{EventSourceTimeout, "<event_source:timeout>"},
			{EventSourceInterval, "<event_source:interval>"},
			{EventSourceTaskDone, "<event_source:task.done>"},
			{EventSourceEOF, "<event_source:eof>"},
			{EventSourceKind(999), "<event_source:unknown>"},
		}

		for _, tt := range tests {
			es := &EventSource{Kind: tt.kind}
			v := EventSourceVal(es)
			if v.String() != tt.want {
				t.Errorf("EventSource(%v).String() = %q, want %q", tt.kind, v.String(), tt.want)
			}
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		v := Value{Type: Type(999), Data: nil}
		if v.String() != "<unknown>" {
			t.Errorf("Unknown.String() = %q, want <unknown>", v.String())
		}
	})
}

// Test Task concurrent access
func TestTaskConcurrentAccess(t *testing.T) {
	task := &Task{ID: 1}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func(i int) {
			defer wg.Done()
			task.SetStatus(TaskStatus(i % 5))
		}(i)

		go func() {
			defer wg.Done()
			task.GetStatus()
		}()

		go func(i int) {
			defer wg.Done()
			task.SetResult(Int(int64(i)))
			task.GetResult()
		}(i)
	}
	wg.Wait()
	// Test passes if no race condition detected (run with -race)
}

func TestTaskSetGetError(t *testing.T) {
	task := &Task{ID: 1}

	if task.GetError() != nil {
		t.Error("new task should have nil error")
	}

	err := &testError{msg: "test error"}
	task.SetError(err)

	got := task.GetError()
	if got == nil || got.Error() != "test error" {
		t.Errorf("GetError() = %v, want 'test error'", got)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
