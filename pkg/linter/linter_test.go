package linter

import (
	"testing"
)

func ruleNames(res Result) []string {
	out := make([]string, 0, len(res.Findings))
	for _, f := range res.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func TestLint_ValidProgram_NoFindings(t *testing.T) {
	program := `
constraint Positive(x) = x > 0;
func add(a: Positive, b: Positive) = a + b;
add(1, 2);
`
	res := Lint(program)
	if len(res.Findings) != 0 {
		t.Fatalf("expected no findings, got %+v", res.Findings)
	}
}

func TestLint_ParseError(t *testing.T) {
	res := Lint("func broken( = ;")
	if len(res.Findings) == 0 {
		t.Fatal("expected parse findings")
	}
	for _, f := range res.Findings {
		if f.Rule != "parse" {
			t.Errorf("expected rule 'parse', got %q", f.Rule)
		}
		if f.Severity != SeverityError {
			t.Errorf("expected error severity, got %v", f.Severity)
		}
	}
	if res.Findings[0].Line < 1 {
		t.Errorf("expected a nonzero line, got %d", res.Findings[0].Line)
	}
}

func TestLint_TypeMismatch(t *testing.T) {
	res := Lint(`x: Int = "hello";`)
	found := false
	for _, f := range res.Findings {
		if f.Rule == "type" {
			found = true
			if f.Line != 1 {
				t.Errorf("expected line 1, got %d", f.Line)
			}
		}
	}
	if !found {
		t.Fatalf("expected a type finding, got rules %v", ruleNames(res))
	}
}

func TestLint_StaticConstraintViolation(t *testing.T) {
	program := `
constraint Positive(x) = x > 0;
func f(a: Positive) = a;
f(0);
`
	res := Lint(program)
	found := false
	for _, f := range res.Findings {
		if f.Rule == "static" {
			found = true
			if f.Line != 4 {
				t.Errorf("expected line 4, got %d", f.Line)
			}
		}
	}
	if !found {
		t.Fatalf("expected a static finding, got rules %v", ruleNames(res))
	}
}

func TestExtractPosition(t *testing.T) {
	line, col, rest := extractPosition("line 12, column 5: oops")
	if line != 12 || col != 5 || rest != "oops" {
		t.Fatalf("got line=%d col=%d rest=%q", line, col, rest)
	}
	line, col, rest = extractPosition("no position here")
	if line != 0 || col != 0 || rest != "no position here" {
		t.Fatalf("got line=%d col=%d rest=%q", line, col, rest)
	}
}
