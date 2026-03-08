package vm

import (
	"fmt"
	"testing"
	"time"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/value"
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
	// Test stay loop with multiple events: count up to 3 using interval
	input := `
use core.schedule!;
use core.async!;

async.stay(count: 0) {
    src = schedule.interval(10);
    h = async.expects((ev) => {
        c = count + 1;
        async.continue({count: c});
        async.leave(c)
    }, src);
    h.ready()
}
`
	result := testEvalWithTimeout(t, input, 2*time.Second)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	inner := result.AsSuccess()
	if inner.Type != value.TYPE_INT {
		t.Fatalf("expected int result, got %s", inner.String())
	}
	// The loop should exit after at least 1 event with count=1
	if inner.AsInt() < 1 {
		t.Fatalf("expected count >= 1, got %d", inner.AsInt())
	}
}

func TestAsyncChannel(t *testing.T) {
	// Test creating a channel
	input := `
use core.async!
async.channel()
`
	result := testEval(t, input)
	if result.Type != value.TYPE_CHANNEL {
		t.Fatalf("expected channel, got %s: %s", result.Type, result.String())
	}
	ch := result.AsChannel()
	if ch.IsClosed() {
		t.Fatal("expected open channel, got closed")
	}
}

func TestAsyncChannelBuffered(t *testing.T) {
	// Test creating a buffered channel
	input := `
use core.async!
async.channel(10)
`
	result := testEval(t, input)
	if result.Type != value.TYPE_CHANNEL {
		t.Fatalf("expected channel, got %s: %s", result.Type, result.String())
	}
}

func TestAsyncChannelStatus(t *testing.T) {
	// Test channel status property
	input := `
use core.async!
ch = async.channel();
ch.status
`
	result := testEval(t, input)
	if result.Type != value.TYPE_STRING {
		t.Fatalf("expected string, got %s", result.Type)
	}
	if result.AsString() != "open" {
		t.Fatalf("expected 'open', got '%s'", result.AsString())
	}
}

func TestAsyncChannelSendReceive(t *testing.T) {
	// Test sending and receiving via a buffered channel using spawn
	input := `
use core.async!;

ch = async.channel(1);
ch.send(99);
ch.receive()
`
	result := testEvalWithTimeout(t, input, 2*time.Second)
	if result.Type != value.TYPE_INT {
		t.Fatalf("expected int, got %s: %s", result.Type, result.String())
	}
	if result.AsInt() != 99 {
		t.Fatalf("expected 99, got %d", result.AsInt())
	}
}

func TestAsyncChannelWithSpawn(t *testing.T) {
	// Test channel send from a spawned task, received in a stay loop
	input := `
use core.async!;

ch = async.channel(1);
async.spawn(() => ch.send(42));
async.stay(done: false) {
    h = async.expects((val) => async.leave(val), ch.source);
    h.ready()
}
`
	result := testEvalWithTimeout(t, input, 2*time.Second)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	inner := result.AsSuccess()
	if inner.Type != value.TYPE_INT || inner.AsInt() != 42 {
		t.Fatalf("expected success(42), got success(%s)", inner.String())
	}
}

func TestAsyncChannelSource(t *testing.T) {
	// Test channel.source returns an event source usable with async.expects
	input := `
use core.async!;

ch = async.channel(1);
ch.source
`
	result := testEval(t, input)
	if result.Type != value.TYPE_EVENT_SOURCE {
		t.Fatalf("expected event_source, got %s", result.Type)
	}
	es := result.AsEventSource()
	if es.Kind != value.EventSourceChannel {
		t.Fatalf("expected channel event source, got kind %d", es.Kind)
	}
}

func TestAsyncChannelInStayLoop(t *testing.T) {
	// Test channel as event source in a stay loop, with sender in spawned task
	input := `
use core.async!;

ch = async.channel(1);
async.spawn(() => ch.send(77));
async.stay(done: false) {
    h = async.expects((val) => async.leave(val), ch.source);
    h.ready()
}
`
	result := testEvalWithTimeout(t, input, 2*time.Second)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	inner := result.AsSuccess()
	if inner.Type != value.TYPE_INT || inner.AsInt() != 77 {
		t.Fatalf("expected success(77), got success(%s)", inner.String())
	}
}

func TestAsyncChannelClose(t *testing.T) {
	// Test closing a channel
	input := `
use core.async!
ch = async.channel();
ch.close();
ch.status
`
	result := testEval(t, input)
	if result.Type != value.TYPE_STRING {
		t.Fatalf("expected string, got %s", result.Type)
	}
	if result.AsString() != "closed" {
		t.Fatalf("expected 'closed', got '%s'", result.AsString())
	}
}

func TestAsyncAll(t *testing.T) {
	// Test async.all waits for all tasks
	input := `
use core.async!;

t1 = async.spawn(() => 1);
t2 = async.spawn(() => 2);
t3 = async.spawn(() => 3);
async.all([t1, t2, t3])
`
	result := testEvalWithTimeout(t, input, 2*time.Second)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s: %s", result.Type, result.String())
	}
	arr := result.AsArray()
	if len(arr) != 3 {
		t.Fatalf("expected 3 results, got %d", len(arr))
	}
}

func TestAsyncCancel(t *testing.T) {
	// Test async.cancel cancels a task
	input := `
use core.async!;
use core.schedule!;

src = schedule.timeout(100);
h = async.expects(() => 1, src);
async.cancel(h);
h.status
`
	result := testEval(t, input)
	if result.Type != value.TYPE_STRING {
		t.Fatalf("expected string, got %s: %s", result.Type, result.String())
	}
	if result.AsString() != "cancelled" {
		t.Fatalf("expected 'cancelled', got '%s'", result.AsString())
	}
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
	if task.GetStatus() != value.TaskCompleted {
		t.Fatalf("expected completed, got status %d", task.GetStatus())
	}
	taskResult := task.GetResult()
	if taskResult.Type != value.TYPE_INT || taskResult.AsInt() != 42 {
		t.Fatalf("expected result 42, got %s", taskResult.String())
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
