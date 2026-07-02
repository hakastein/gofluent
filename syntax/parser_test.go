package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

func TestParseSimpleMessage(t *testing.T) {
	res := syntax.Parse("foo = Bar\n")
	require.Len(t, res.Body, 1)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	assert.Equal(t, "foo", msg.ID.Name)

	require.NotNil(t, msg.Value)
	require.Len(t, msg.Value.Elements, 1)

	te, ok := msg.Value.Elements[0].(*ast.TextElement)
	require.True(t, ok, "expected *ast.TextElement, got %T", msg.Value.Elements[0])
	assert.Equal(t, "Bar", te.Value)
}

func TestParseTerm(t *testing.T) {
	res := syntax.Parse("-brand = Firefox\n")
	require.NotEmpty(t, res.Body)

	term, ok := res.Body[0].(*ast.Term)
	require.True(t, ok, "expected *ast.Term, got %T", res.Body[0])
	assert.Equal(t, "brand", term.ID.Name)
}

func TestParseAttributes(t *testing.T) {
	res := syntax.Parse("foo = Bar\n    .attr = Baz\n")
	require.NotEmpty(t, res.Body)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.Len(t, msg.Attributes, 1)
	assert.Equal(t, "attr", msg.Attributes[0].ID.Name)
}

func TestParseSelectExpression(t *testing.T) {
	res := syntax.Parse("foo = { $n ->\n    [one] One\n   *[other] Other\n}\n")
	require.NotEmpty(t, res.Body)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
	require.NotEmpty(t, msg.Value.Elements)

	pl, ok := msg.Value.Elements[0].(*ast.Placeable)
	require.True(t, ok, "expected *ast.Placeable, got %T", msg.Value.Elements[0])

	sel, ok := pl.Expression.(*ast.SelectExpression)
	require.True(t, ok, "expected *ast.SelectExpression, got %T", pl.Expression)

	require.Len(t, sel.Variants, 2)
	assert.True(t, sel.Variants[1].Default, "second variant should be default")

	vr, ok := sel.Selector.(*ast.VariableReference)
	require.True(t, ok, "expected *ast.VariableReference selector, got %T", sel.Selector)
	assert.Equal(t, "n", vr.ID.Name)
}

func TestParseMultilinePattern(t *testing.T) {
	res := syntax.Parse("foo =\n    Line one\n    Line two\n")
	require.NotEmpty(t, res.Body)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
	require.NotEmpty(t, msg.Value.Elements)

	te, ok := msg.Value.Elements[0].(*ast.TextElement)
	require.True(t, ok, "expected *ast.TextElement, got %T", msg.Value.Elements[0])
	assert.Equal(t, "Line one\nLine two", te.Value)
}

func TestParseCallArguments(t *testing.T) {
	res := syntax.Parse("foo = { FUN($a, key: 1) }\n")
	require.NotEmpty(t, res.Body)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
	require.NotEmpty(t, msg.Value.Elements)

	pl, ok := msg.Value.Elements[0].(*ast.Placeable)
	require.True(t, ok, "expected *ast.Placeable, got %T", msg.Value.Elements[0])

	fn, ok := pl.Expression.(*ast.FunctionReference)
	require.True(t, ok, "expected *ast.FunctionReference, got %T", pl.Expression)

	require.Len(t, fn.Arguments.Positional, 1)
	require.Len(t, fn.Arguments.Named, 1)
	assert.Equal(t, "key", fn.Arguments.Named[0].Name.Name)
}

func TestJunkRecovery(t *testing.T) {
	res := syntax.Parse("err = {1xx}\ngood = Value\n")
	require.Len(t, res.Body, 2)

	junk, ok := res.Body[0].(*ast.Junk)
	require.True(t, ok, "expected *ast.Junk, got %T", res.Body[0])
	require.NotEmpty(t, junk.Annotations)
	codes := make([]string, 0, len(junk.Annotations))
	for _, a := range junk.Annotations {
		codes = append(codes, a.Code)
	}
	assert.Contains(t, codes, "E0003")

	_, ok = res.Body[1].(*ast.Message)
	assert.True(t, ok, "expected second entry to be *ast.Message, got %T", res.Body[1])
}

func TestParseEntry(t *testing.T) {
	entry := syntax.ParseEntry("# comment\nfoo = Bar\n")

	msg, ok := entry.(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", entry)
	assert.Equal(t, "foo", msg.ID.Name)
}

func TestCommentAttachment(t *testing.T) {
	res := syntax.Parse("# Hello\nfoo = Bar\n")
	require.Len(t, res.Body, 1)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Comment)
	assert.Equal(t, "Hello", msg.Comment.Content)
}

func TestCommentAstralCharacters(t *testing.T) {
	res := syntax.Parse("# \U0001F600 emoji\nfoo = Bar\n")
	require.Len(t, res.Body, 1)

	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Comment)
	assert.Equal(t, "\U0001F600 emoji", msg.Comment.Content)
}

func TestWithSpansDisabled(t *testing.T) {
	res := syntax.Parse("foo = Bar\n", syntax.WithSpans(false))
	assert.Nil(t, res.GetSpan(), "expected no span on resource")

	require.NotEmpty(t, res.Body)
	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	assert.Nil(t, msg.GetSpan(), "expected no span on message")
}

// TestParseErrorAnnotations verifies that parse errors surface through the
// public contract: an invalid entry becomes ast.Junk carrying an ast.Annotation
// with the right Fluent diagnostic code. The human-readable message wording is
// not part of the contract.
func TestParseErrorAnnotations(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
	}{
		{name: "E0002 expected entry start", src: "123\n", code: "E0002"},
		{name: "E0011 no variant after arrow", src: "foo = { $n ->\n}\n", code: "E0011"},
		{name: "E0028 expected inline expression", src: "foo = { }\n", code: "E0028"},
		{name: "E0003 expected closing brace token", src: "foo = { $n\n", code: "E0003"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := syntax.Parse(tc.src)

			var codes []string
			for _, entry := range res.Body {
				if junk, ok := entry.(*ast.Junk); ok {
					for _, a := range junk.Annotations {
						codes = append(codes, a.Code)
					}
				}
			}
			assert.Containsf(t, codes, tc.code, "expected an annotation for %q", tc.src)
		})
	}
}
