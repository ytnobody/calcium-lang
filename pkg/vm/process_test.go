package vm

import (
	"os/exec"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

func TestProcStreamEcho(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available on this platform")
	}

	// Launch a long-lived process. `cat` echoes stdin back to stdout, which is
	// a convenient stand-in for a DBI backend speaking over a text stream.
	start := primitiveProcStart(value.String("cat"), value.Array([]value.Value{}))
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()
	id := int(handle.AsInt())
	if id < 1 {
		t.Fatalf("unexpected handle: %v", id)
	}

	// Write one line and read it back.
	w := primitiveProcWrite(handle, value.String("hello\n"))
	if !w.IsSuccess() {
		t.Fatalf("__proc_write failed: %v", w)
	}
	line := primitiveProcReadLine(value.Int(int64(id)))
	if !line.IsSuccess() {
		t.Fatalf("__proc_read_line failed: %v", line)
	}
	if got := line.AsSuccess().AsString(); got != "hello" {
		t.Fatalf("expected echo 'hello', got %q", got)
	}

	// Close terminates and reaps the child.
	c := primitiveProcClose(handle)
	if !c.IsSuccess() {
		t.Fatalf("__proc_close failed: %v", c)
	}
}

func TestProcInvalidHandle(t *testing.T) {
	bad := value.Int(99999)
	if r := primitiveProcWrite(bad, value.String("x")); r.IsSuccess() {
		t.Fatal("expected failure for unknown handle")
	}
	if r := primitiveProcReadLine(bad); r.IsSuccess() {
		t.Fatal("expected failure for unknown handle")
	}
	if r := primitiveProcClose(value.String("not-an-int")); r.IsSuccess() {
		t.Fatal("expected failure for non-integer handle")
	}
}

func TestKillAllUnregisters(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available on this platform")
	}
	start := primitiveProcStart(value.String("sleep"), value.Array([]value.Value{value.String("30")}))
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()

	KillAll()

	// After KillAll the process is gone from the registry.
	r := primitiveProcClose(handle) // should produce success (already removed => success)
	if !r.IsSuccess() {
		t.Fatalf("__proc_close after KillAll failed: %v", r)
	}
}
