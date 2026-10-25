package parser_test

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
)

func parseComposite(t *testing.T, src string) *ast.FunctionDeclaration {
	t.Helper()
	p := parser.New(lexer.New(src))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	fd, ok := prog.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", prog.Statements[0])
	}
	return fd
}

func TestParseCompositeAnnotation_AND(t *testing.T) {
	fd := parseComposite(t, `func f(a: Positive & NonZero) = a;`)
	if len(fd.CompositeConstraints) != 1 {
		t.Fatalf("expected 1 composite, got %d", len(fd.CompositeConstraints))
	}
	c := fd.CompositeConstraints[0]
	if c.Operator != "&" {
		t.Errorf("expected operator '&', got %q", c.Operator)
	}
	if len(c.Parts) != 2 || c.Parts[0].Value != "Positive" || c.Parts[1].Value != "NonZero" {
		t.Errorf("unexpected parts: %+v", c.Parts)
	}
}

func TestParseCompositeAnnotation_OR(t *testing.T) {
	fd := parseComposite(t, `func f(a: Int | String) = a;`)
	c := fd.CompositeConstraints[0]
	if c.Operator != "|" {
		t.Errorf("expected operator '|', got %q", c.Operator)
	}
	if len(c.Parts) != 2 || c.Parts[0].Value != "Int" || c.Parts[1].Value != "String" {
		t.Errorf("unexpected parts: %+v", c.Parts)
	}
}

func TestParseCompositeAnnotation_ThreeParts(t *testing.T) {
	fd := parseComposite(t, `func f(a: Int & Positive & NonZero) = a;`)
	c := fd.CompositeConstraints[0]
	if len(c.Parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(c.Parts))
	}
}

func TestParseCompositeAnnotation_MixedOperatorRejected(t *testing.T) {
	p := parser.New(lexer.New(`func f(a: Int & String | Maybe) = a;`))
	p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected parse error for mixing '&' and '|'")
	}
}
