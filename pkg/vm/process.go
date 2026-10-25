package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// ---------------------------------------------------------------------------
// External process management
//
// The VM can launch long-lived external programs (e.g. a DBI backend) and talk
// to them over their stdin/stdout as a text stream. Every spawned process is
// tracked so it can be reliably terminated when the parent exits, preventing
// orphaned children.
//
// Children are launched in the same process group as the parent (the default),
// so terminal-delivered signals (e.g. Ctrl+C / SIGINT to the foreground group)
// propagate naturally to them. In addition, KillAll() is invoked on program /
// REPL teardown and inside __os_exit so all children are reaped on normal,
// resource-exiting, and error paths.
// ---------------------------------------------------------------------------

// proc is a tracked, running external process.
type proc struct {
	id     int
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	closed bool

	waitOnce sync.Once
	exitCode int
}

var (
	procMu   sync.Mutex
	procs    = map[int]*proc{}
	nextProc = 1
)

// KillAll terminates every currently tracked subprocess. It is safe to call
// multiple times and with no tracked processes. Called on program/REPL teardown
// and from __os_exit so children do not outlive their parent.
func KillAll() {
	procMu.Lock()
	all := make([]*proc, 0, len(procs))
	for _, p := range procs {
		all = append(all, p)
	}
	procs = map[int]*proc{}
	procMu.Unlock()

	for _, p := range all {
		p.terminate()
	}
}

// terminate forcefully stops a child process and reaps it.
func (p *proc) terminate() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	if p.stdin != nil {
		p.stdin.Close() //nolint:errcheck
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill() //nolint:errcheck
	}
	p.wait()
}

// wait blocks until the process exits and records its exit code (called once).
func (p *proc) wait() {
	p.waitOnce.Do(func() {
		err := p.cmd.Wait()
		if ee, ok := err.(*exec.ExitError); ok {
			p.exitCode = ee.ExitCode()
		}
	})
}

// lookup returns the tracked process for an id, or nil.
func procLookup(id int) *proc {
	procMu.Lock()
	defer procMu.Unlock()
	return procs[id]
}

// procRegister adds a process to the registry under a fresh id.
func procRegister(p *proc) {
	procMu.Lock()
	defer procMu.Unlock()
	p.id = nextProc
	procs[nextProc] = p
	nextProc++
}

// procRemove deletes a process from the registry.
func procRemove(p *proc) {
	procMu.Lock()
	defer procMu.Unlock()
	delete(procs, p.id)
}

// ---------------------------------------------------------------------------
// Primitives
// ---------------------------------------------------------------------------

// __proc_start(cmd, argsArray) -> success(handle) | failure(msg)
// Launches a long-lived external program and returns an integer handle for
// talking to it over stdin/stdout.
func primitiveProcStart(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__proc_start: expected 2 arguments (command, args array)"))
	}
	if args[0].Type != value.TYPE_STRING {
		return value.Failure(value.String("__proc_start: command must be string"))
	}
	if args[1].Type != value.TYPE_ARRAY {
		return value.Failure(value.String("__proc_start: args must be an array of strings"))
	}

	cmdName := args[0].AsString()
	argVals := args[1].AsArray()
	argv := make([]string, 0, len(argVals))
	for _, v := range argVals {
		if v.Type != value.TYPE_STRING {
			return value.Failure(value.String("__proc_start: args must all be strings"))
		}
		argv = append(argv, v.AsString())
	}

	cmd := exec.Command(cmdName, argv...)
	cmd.Stderr = os.Stderr // protocol lives on stdout; logs go to our stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__proc_start: %v", err)))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return value.Failure(value.String(fmt.Sprintf("__proc_start: %v", err)))
	}

	if err := cmd.Start(); err != nil {
		return value.Failure(value.String(fmt.Sprintf("__proc_start: %v", err)))
	}

	p := &proc{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
	procRegister(p)
	return value.Success(value.Int(int64(p.id)))
}

// __proc_write(handle, string) -> success(true) | failure(msg)
func primitiveProcWrite(args ...value.Value) value.Value {
	if len(args) != 2 {
		return value.Failure(value.String("__proc_write: expected 2 arguments (handle, string)"))
	}
	handle, ok := procHandleArg(args[0])
	if !ok {
		return value.Failure(value.String("__proc_write: invalid handle"))
	}
	p := procLookup(handle)
	if p == nil {
		return value.Failure(value.String("__proc_write: no such process handle"))
	}
	if args[1].Type != value.TYPE_STRING {
		return value.Failure(value.String("__proc_write: data must be string"))
	}
	if _, err := io.WriteString(p.stdin, args[1].AsString()); err != nil {
		return value.Failure(value.String(fmt.Sprintf("__proc_write: %v", err)))
	}
	return value.Success(value.Bool(true))
}

// __proc_write_secret(handle, secret, capability) -> success(true) | failure(msg)
// Trusted sink: verifies the capability and writes the Secret payload to the
// child process stdin without exposing the raw value to Calcium code.
func primitiveProcWriteSecret(args ...value.Value) value.Value {
	if len(args) != 3 {
		return value.Failure(value.String("__proc_write_secret: expected 3 arguments (handle, secret, capability)"))
	}
	handle, ok := procHandleArg(args[0])
	if !ok {
		return value.Failure(value.String("__proc_write_secret: invalid handle"))
	}
	if args[1].Type != value.TYPE_SECRET {
		return value.Failure(value.String("__proc_write_secret: secret must be a Secret"))
	}
	if args[2].Type != value.TYPE_SECRET_CAPABILITY {
		return value.Failure(value.String("__proc_write_secret: capability must be a SecretCapability"))
	}
	cap := args[2].AsSecretCapability()
	if cap.Sink != "os.proc_write_secret" {
		return value.Failure(value.String(fmt.Sprintf("__proc_write_secret: capability authorizes %q, not os.proc_write_secret", cap.Sink)))
	}
	if cap.Scope.Type != value.TYPE_HASH {
		return value.Failure(value.String("__proc_write_secret: capability scope must include a handle"))
	}
	scopeHandle, found := cap.Scope.AsHash().Get(value.String("handle"))
	if !found || !scopeHandle.Equals(args[0]) {
		return value.Failure(value.String("__proc_write_secret: capability does not authorize this process handle"))
	}
	payload := args[1].AsSecret().Value
	if payload.Type != value.TYPE_STRING {
		return value.Failure(value.String("__proc_write_secret: Secret payload must be a string"))
	}
	p := procLookup(handle)
	if p == nil {
		return value.Failure(value.String("__proc_write_secret: no such process handle"))
	}
	if _, err := io.WriteString(p.stdin, payload.AsString()); err != nil {
		return value.Failure(value.String(fmt.Sprintf("__proc_write_secret: %v", err)))
	}
	return value.Success(value.Bool(true))
}

// __proc_read_line(handle) -> success(string) | failure(msg)
// Reads one line (terminated by '\n') from the process stdout. Blocks until
// data is available or the process closes its stdout.
func primitiveProcReadLine(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__proc_read_line: expected 1 argument (handle)"))
	}
	handle, ok := procHandleArg(args[0])
	if !ok {
		return value.Failure(value.String("__proc_read_line: invalid handle"))
	}
	p := procLookup(handle)
	if p == nil {
		return value.Failure(value.String("__proc_read_line: no such process handle"))
	}
	line, err := p.stdout.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return value.Failure(value.String("__proc_read_line: process closed"))
		}
		return value.Failure(value.String(fmt.Sprintf("__proc_read_line: %v", err)))
	}
	line = strings.TrimSuffix(line, "\n")
	return value.Success(value.String(line))
}

// __proc_close(handle) -> success(true) | failure(msg)
// Forcefully stops the process (and reaps it) if still running.
func primitiveProcClose(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__proc_close: expected 1 argument (handle)"))
	}
	handle, ok := procHandleArg(args[0])
	if !ok {
		return value.Failure(value.String("__proc_close: invalid handle"))
	}
	p := procLookup(handle)
	if p == nil {
		return value.Success(value.Bool(true))
	}
	procRemove(p)
	p.terminate()
	return value.Success(value.Bool(true))
}

// __proc_wait(handle) -> success(exitCode) | failure(msg)
// Blocks until the process exits and returns its exit code.
func primitiveProcWait(args ...value.Value) value.Value {
	if len(args) != 1 {
		return value.Failure(value.String("__proc_wait: expected 1 argument (handle)"))
	}
	handle, ok := procHandleArg(args[0])
	if !ok {
		return value.Failure(value.String("__proc_wait: invalid handle"))
	}
	p := procLookup(handle)
	if p == nil {
		return value.Failure(value.String("__proc_wait: no such process handle"))
	}
	p.wait()
	procRemove(p)
	return value.Success(value.Int(int64(p.exitCode)))
}

// procHandleArg validates and extracts an integer handle value.
func procHandleArg(v value.Value) (int, bool) {
	if v.Type != value.TYPE_INT {
		return 0, false
	}
	return int(v.AsInt()), true
}
