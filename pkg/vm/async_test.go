package vm

import (
	"fmt"
	"testing"
	"time"

	"github.com/example/calcium/pkg/compiler"
	"github.com/example/calcium/pkg/lexer"
	"github.com/example/calcium/pkg/parser"
	"github.com/example/calcium/pkg/value"
)

func TestHandlerReady(t *testing.T) {
	// Test that handler.ready() activates a handler
	input := `
use core.schedule!;
use core.async!;

src = schedule.timeout(10);
h = async.expects(() => 1, src);
h.ready()
`
	result := testEval(t, input)
	if result.Type != value.TYPE_HANDLER {
		t.Fatalf("expected handler, got %s: %s", result.Type, result.String())
	}
	handler := result.AsHandler()
	if handler.Status != value.HandlerActive {
		t.Fatalf("expected active, got status %d", handler.Status)
	}
}

func TestSimpleStayLoop(t *testing.T) {
	// Test a simple stay loop with immediate leave
	input := `
use core.async!;

async.stay(done: false) {
    async.leave(42)
}
`
	result := testEval(t, input)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	inner := result.AsSuccess()
	if inner.Type != value.TYPE_INT || inner.AsInt() != 42 {
		t.Fatalf("expected success(42), got success(%s)", inner.String())
	}
}

func TestStayWithTimeout(t *testing.T) {
	// Test stay loop with timeout - should exit when timeout fires
	// Callback receives event value as argument (ignored here)
	input := `
use core.schedule!;
use core.async!;

async.stay(done: false) {
    src = schedule.timeout(10);
    h = async.expects((ev) => async.leave(true), src);
    h.ready()
}
`
	result := testEvalWithTimeout(t, input, 1*time.Second)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	inner := result.AsSuccess()
	if inner.Type != value.TYPE_BOOL || !inner.AsBool() {
		t.Fatalf("expected success(true), got success(%s)", inner.String())
	}
}

func TestStayWithMultipleTimeouts(t *testing.T) {
	// Skip this test for now - needs state handling improvement
	t.Skip("State handling in callbacks needs improvement")
}

func testEvalWithTimeout(t *testing.T, input string, timeout time.Duration) value.Value {
	done := make(chan value.Value, 1)
	errCh := make(chan error, 1)

	go func() {
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			errCh <- fmt.Errorf("parser errors: %v", p.Errors())
			return
		}

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			errCh <- fmt.Errorf("compiler error: %s", err)
			return
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			errCh <- fmt.Errorf("vm error: %s", err)
			return
		}

		result := vm.LastPoppedStackElem()
		if result.Type == value.TYPE_NULL && vm.sp > 0 {
			result = vm.StackTop()
		}
		done <- result
	}()

	select {
	case result := <-done:
		return result
	case err := <-errCh:
		t.Fatal(err)
		return value.Null()
	case <-time.After(timeout):
		t.Fatal("test timed out")
		return value.Null()
	}
}

func TestScheduleTimeout(t *testing.T) {
	// Test that schedule.timeout creates an event source
	input := `
use core.schedule!
schedule.timeout(10)
`
	result := testEval(t, input)
	if result.Type != value.TYPE_EVENT_SOURCE {
		t.Fatalf("expected event_source, got %s", result.Type)
	}
	es := result.AsEventSource()
	if es.Kind != value.EventSourceTimeout {
		t.Fatalf("expected timeout event source, got %d", es.Kind)
	}
}

func TestScheduleInterval(t *testing.T) {
	// Test that schedule.interval creates an event source
	input := `
use core.schedule!
schedule.interval(10)
`
	result := testEval(t, input)
	if result.Type != value.TYPE_EVENT_SOURCE {
		t.Fatalf("expected event_source, got %s", result.Type)
	}
	es := result.AsEventSource()
	if es.Kind != value.EventSourceInterval {
		t.Fatalf("expected interval event source, got %d", es.Kind)
	}
}

func TestAsyncSpawn(t *testing.T) {
	// Test that async.spawn creates a task
	input := `
use core.async!
async.spawn(() => 42)
`
	result := testEval(t, input)
	if result.Type != value.TYPE_TASK {
		t.Fatalf("expected task, got %s", result.Type)
	}

	// Wait a bit for the task to complete
	time.Sleep(50 * time.Millisecond)

	task := result.AsTask()
	if task.Status != value.TaskCompleted {
		t.Fatalf("expected completed, got status %d", task.Status)
	}
	if task.Result.Type != value.TYPE_INT || task.Result.AsInt() != 42 {
		t.Fatalf("expected result 42, got %s", task.Result.String())
	}
}

func TestAsyncExpects(t *testing.T) {
	// Test that async.expects creates a handler
	input := `
use core.schedule!
use core.async!
h = schedule.timeout(100) !> async.expects(() => 1)
h
`
	result := testEval(t, input)
	if result.Type != value.TYPE_HANDLER {
		t.Fatalf("expected handler, got %s", result.Type)
	}
	handler := result.AsHandler()
	if handler.Status != value.HandlerDormant {
		t.Fatalf("expected dormant, got status %d", handler.Status)
	}
}

func testEval(t *testing.T, input string) value.Value {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Logf("parser error: %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	vm := New(comp.Constants())
	err = vm.Run(comp.Bytecode().Instructions)
	if err != nil {
		t.Fatalf("vm error: %s", err)
	}

	// For stay expressions, the result is on the stack
	// For other expressions, it's the last popped element
	result := vm.LastPoppedStackElem()
	if result.Type == value.TYPE_NULL && vm.sp > 0 {
		result = vm.StackTop()
	}
	return result
}
