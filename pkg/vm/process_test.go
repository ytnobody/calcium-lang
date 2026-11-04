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

func TestProcWriteSecretTrustedSink(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available on this platform")
	}

	start := primitiveProcStart(value.String("cat"), value.Array([]value.Value{}))
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()
	defer primitiveProcClose(handle) //nolint:errcheck

	scope := value.NewHash()
	scope.Set(value.String("handle"), handle)
	cap := value.SecretCapabilityVal("os.proc_write_secret", value.HashVal(scope))

	write := primitiveProcWriteSecret(handle, value.SecretVal(value.String("SECRET-LINE\n")), cap)
	if !write.IsSuccess() {
		t.Fatalf("__proc_write_secret failed: %v", write)
	}

	line := primitiveProcReadLine(handle)
	if !line.IsSuccess() {
		t.Fatalf("__proc_read_line failed: %v", line)
	}
	if got := line.AsSuccess().AsString(); got != "SECRET-LINE" {
		t.Fatalf("expected SECRET-LINE, got %q", got)
	}
}

func TestProcWriteSecretRejectsWrongHandleScope(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available on this platform")
	}

	start := primitiveProcStart(value.String("cat"), value.Array([]value.Value{}))
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()
	defer primitiveProcClose(handle) //nolint:errcheck

	scope := value.NewHash()
	scope.Set(value.String("handle"), value.Int(99999))
	cap := value.SecretCapabilityVal("os.proc_write_secret", value.HashVal(scope))

	write := primitiveProcWriteSecret(handle, value.SecretVal(value.String("SECRET")), cap)
	if write.IsSuccess() {
		t.Fatal("expected failure for mismatched handle scope")
	}
}

// The child must not inherit this process's environment: a Calcium program commonly
// runs where credentials live in the environment, and `Secret` cannot protect what
// never entered the language.
func TestProcStartDoesNotInheritEnvironment(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available on this platform")
	}
	t.Setenv("CALCIUM_LEAK_CANARY", "secret-value")

	start := primitiveProcStart(
		value.String(sh),
		value.Array([]value.Value{value.String("-c"), value.String("echo \"${CALCIUM_LEAK_CANARY-unset}\"")}),
	)
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()
	line := primitiveProcReadLine(handle)
	if !line.IsSuccess() {
		t.Fatalf("__proc_read_line failed: %v", line)
	}
	if got := line.AsSuccess().AsString(); got != "unset" {
		t.Fatalf("the child saw the parent's environment: %q", got)
	}
	primitiveProcClose(handle)
}

// With an explicit env hash, exactly the given names are passed.
func TestProcStartPassesOnlyTheGivenEnvironment(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available on this platform")
	}
	t.Setenv("CALCIUM_LEAK_CANARY", "secret-value")

	h := value.NewHash()
	h.Set(value.String("CALCIUM_PASSED"), value.String("visible"))
	env := value.HashVal(h)

	start := primitiveProcStart(
		value.String(sh),
		value.Array([]value.Value{
			value.String("-c"),
			value.String("echo \"${CALCIUM_PASSED-unset}/${CALCIUM_LEAK_CANARY-unset}\""),
		}),
		env,
	)
	if !start.IsSuccess() {
		t.Fatalf("__proc_start failed: %v", start)
	}
	handle := start.AsSuccess()
	line := primitiveProcReadLine(handle)
	if !line.IsSuccess() {
		t.Fatalf("__proc_read_line failed: %v", line)
	}
	if got := line.AsSuccess().AsString(); got != "visible/unset" {
		t.Fatalf("expected only the passed variable, got %q", got)
	}
	primitiveProcClose(handle)
}
