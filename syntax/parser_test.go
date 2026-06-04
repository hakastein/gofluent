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
	require.Len(t, junk.Annotations, 1)
	assert.Equal(t, "E0003", junk.Annotations[0].Code)

	_, ok = res.Body[1].(*ast.Message)
	assert.True(t, ok, "expected second entry to be *ast.Message, got %T", res.Body[1])
}

func TestParseEntry(t *testing.T) {
	entry, err := syntax.ParseEntry("# comment\nfoo = Bar\n")
	require.NoError(t, err)

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
// whose Code and Message match the Fluent diagnostic. This covers the
// error-message formatting that callers actually observe (the parser's internal
// message table is exercised end-to-end here rather than via white-box access).
func TestParseErrorAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		code    string
		message string
	}{
		{
			name:    "E0002 expected entry start",
			src:     "123\n",
			code:    "E0002",
			message: "Expected an entry start",
		},
		{
			name:    "E0011 no variant after arrow",
			src:     "foo = { $n ->\n}\n",
			code:    "E0011",
			message: `Expected at least one variant after "->"`,
		},
		{
			name:    "E0028 expected inline expression",
			src:     "foo = { }\n",
			code:    "E0028",
			message: "Expected an inline expression",
		},
		{
			name:    "E0003 expected closing brace token",
			src:     "foo = { $n\n",
			code:    "E0003",
			message: `Expected token: "}"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := syntax.Parse(tc.src)

			var ann *ast.Annotation
			for _, entry := range res.Body {
				if junk, ok := entry.(*ast.Junk); ok && len(junk.Annotations) > 0 {
					ann = junk.Annotations[0]
					break
				}
			}
			require.NotNil(t, ann, "expected a Junk annotation for %q", tc.src)
			assert.Equal(t, tc.code, ann.Code)
			assert.Equal(t, tc.message, ann.Message)
		})
	}
}
