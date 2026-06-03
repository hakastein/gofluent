package syntax

import (
	"testing"

	"github.com/hakastein/gofluent/syntax/ast"
)

func TestParseSimpleMessage(t *testing.T) {
	res := Parse("foo = Bar\n")
	if len(res.Body) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(res.Body))
	}
	msg, ok := res.Body[0].(*ast.Message)
	if !ok {
		t.Fatalf("expected Message, got %T", res.Body[0])
	}
	if msg.ID.Name != "foo" {
		t.Errorf("id = %q, want foo", msg.ID.Name)
	}
	if msg.Value == nil || len(msg.Value.Elements) != 1 {
		t.Fatalf("unexpected value: %+v", msg.Value)
	}
	te, ok := msg.Value.Elements[0].(*ast.TextElement)
	if !ok || te.Value != "Bar" {
		t.Errorf("text element = %+v, want Bar", msg.Value.Elements[0])
	}
}

func TestParseTerm(t *testing.T) {
	res := Parse("-brand = Firefox\n")
	term, ok := res.Body[0].(*ast.Term)
	if !ok {
		t.Fatalf("expected Term, got %T", res.Body[0])
	}
	if term.ID.Name != "brand" {
		t.Errorf("id = %q, want brand", term.ID.Name)
	}
}

func TestParseAttributes(t *testing.T) {
	res := Parse("foo = Bar\n    .attr = Baz\n")
	msg := res.Body[0].(*ast.Message)
	if len(msg.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(msg.Attributes))
	}
	if msg.Attributes[0].ID.Name != "attr" {
		t.Errorf("attr id = %q, want attr", msg.Attributes[0].ID.Name)
	}
}

func TestParseSelectExpression(t *testing.T) {
	src := "foo = { $n ->\n    [one] One\n   *[other] Other\n}\n"
	res := Parse(src)
	msg := res.Body[0].(*ast.Message)
	pl := msg.Value.Elements[0].(*ast.Placeable)
	sel, ok := pl.Expression.(*ast.SelectExpression)
	if !ok {
		t.Fatalf("expected SelectExpression, got %T", pl.Expression)
	}
	if len(sel.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(sel.Variants))
	}
	if !sel.Variants[1].Default {
		t.Errorf("second variant should be default")
	}
	if vr, ok := sel.Selector.(*ast.VariableReference); !ok || vr.ID.Name != "n" {
		t.Errorf("selector = %+v, want $n", sel.Selector)
	}
}

func TestParseMultilinePattern(t *testing.T) {
	src := "foo =\n    Line one\n    Line two\n"
	res := Parse(src)
	msg := res.Body[0].(*ast.Message)
	te := msg.Value.Elements[0].(*ast.TextElement)
	if te.Value != "Line one\nLine two" {
		t.Errorf("multiline value = %q", te.Value)
	}
}

func TestParseCallArguments(t *testing.T) {
	src := "foo = { FUN($a, key: 1) }\n"
	res := Parse(src)
	msg := res.Body[0].(*ast.Message)
	pl := msg.Value.Elements[0].(*ast.Placeable)
	fn, ok := pl.Expression.(*ast.FunctionReference)
	if !ok {
		t.Fatalf("expected FunctionReference, got %T", pl.Expression)
	}
	if len(fn.Arguments.Positional) != 1 || len(fn.Arguments.Named) != 1 {
		t.Fatalf("args = %+v", fn.Arguments)
	}
	if fn.Arguments.Named[0].Name.Name != "key" {
		t.Errorf("named arg = %q", fn.Arguments.Named[0].Name.Name)
	}
}

func TestJunkRecovery(t *testing.T) {
	src := "err = {1xx}\ngood = Value\n"
	res := Parse(src)
	if len(res.Body) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(res.Body))
	}
	junk, ok := res.Body[0].(*ast.Junk)
	if !ok {
		t.Fatalf("expected Junk, got %T", res.Body[0])
	}
	if len(junk.Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(junk.Annotations))
	}
	if junk.Annotations[0].Code != "E0003" {
		t.Errorf("annotation code = %q, want E0003", junk.Annotations[0].Code)
	}
	if _, ok := res.Body[1].(*ast.Message); !ok {
		t.Errorf("expected second entry to be Message, got %T", res.Body[1])
	}
}

func TestParseEntry(t *testing.T) {
	entry, err := ParseEntry("# comment\nfoo = Bar\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := entry.(*ast.Message)
	if !ok {
		t.Fatalf("expected Message, got %T", entry)
	}
	if msg.ID.Name != "foo" {
		t.Errorf("id = %q, want foo", msg.ID.Name)
	}
}

func TestCommentAttachment(t *testing.T) {
	res := Parse("# Hello\nfoo = Bar\n")
	if len(res.Body) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(res.Body))
	}
	msg := res.Body[0].(*ast.Message)
	if msg.Comment == nil || msg.Comment.Content != "Hello" {
		t.Errorf("comment = %+v", msg.Comment)
	}
}

func TestWithSpansDisabled(t *testing.T) {
	res := Parse("foo = Bar\n", WithSpans(false))
	if res.GetSpan() != nil {
		t.Errorf("expected no span on resource")
	}
	msg := res.Body[0].(*ast.Message)
	if msg.GetSpan() != nil {
		t.Errorf("expected no span on message")
	}
}

func TestErrorMessages(t *testing.T) {
	cases := map[string]string{
		"E0002": "Expected an entry start",
		"E0011": "Expected at least one variant after \"->\"",
		"E0028": "Expected an inline expression",
	}
	for code, want := range cases {
		if got := getErrorMessage(code, nil); got != want {
			t.Errorf("getErrorMessage(%s) = %q, want %q", code, got, want)
		}
	}
	if got := getErrorMessage("E0003", []interface{}{"}"}); got != `Expected token: "}"` {
		t.Errorf("E0003 = %q", got)
	}
}
