package eval_test

import (
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/eval"
)

func TestEval_HelloWorld(t *testing.T) {
	src := `use core.io!;
"Hello, World!" !> io.say;`
	result := eval.Eval(src)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "Hello, World!") {
		t.Errorf("expected output to contain 'Hello, World!', got %q", result.Output)
	}
}

func TestEval_Print(t *testing.T) {
	src := `use core.io!;
io.print("foo");
io.print("bar");`
	result := eval.Eval(src)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Output != "foobar" {
		t.Errorf("expected 'foobar', got %q", result.Output)
	}
}

func TestEval_Arithmetic(t *testing.T) {
	src := `use core.io!;
x = 6 * 7;
to_string(x) !> io.say;`
	result := eval.Eval(src)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "42") {
		t.Errorf("expected output to contain '42', got %q", result.Output)
	}
}

func TestEval_ParseError(t *testing.T) {
	src := `use core.io!;
this is invalid !!!`
	result := eval.Eval(src)
	if result.Error == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestEval_MultipleLines(t *testing.T) {
	src := `use core.io!;
io.println("line1");
io.println("line2");
io.println("line3");`
	result := eval.Eval(src)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	lines := strings.Split(strings.TrimRight(result.Output, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), result.Output)
	}
}

func TestEvalInto_WritesToWriter(t *testing.T) {
	var sb strings.Builder
	src := `use core.io!;
"written to writer" !> io.say;`
	err := eval.EvalInto(src, &sb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sb.String(), "written to writer") {
		t.Errorf("expected writer to contain 'written to writer', got %q", sb.String())
	}
}
