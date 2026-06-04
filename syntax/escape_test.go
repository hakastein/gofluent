package syntax_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

// firstStringLiteral parses src and returns the first StringLiteral it finds in
// the first message's value.
func firstStringLiteral(t *testing.T, src string) *ast.StringLiteral {
	t.Helper()
	res := syntax.Parse(src)
	require.NotEmpty(t, res.Body)
	msg, ok := res.Body[0].(*ast.Message)
	require.True(t, ok, "expected *ast.Message, got %T", res.Body[0])
	require.NotNil(t, msg.Value)
	require.NotEmpty(t, msg.Value.Elements)
	pl, ok := msg.Value.Elements[0].(*ast.Placeable)
	require.True(t, ok, "expected *ast.Placeable, got %T", msg.Value.Elements[0])
	sl, ok := pl.Expression.(*ast.StringLiteral)
	require.True(t, ok, "expected *ast.StringLiteral, got %T", pl.Expression)
	return sl
}

// TestStringLiteralParseHexEscape checks decoding of unicode escapes through the
// public StringLiteral.Parse API, including out-of-range code points which must
// collapse to U+FFFD (matching the reference's string([]rune{...}) behavior).
func TestStringLiteralParseHexEscape(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "bmp", src: `k = {"A"}` + "\n", want: "A"},
		{name: "astral", src: `k = {"\U01F600"}` + "\n", want: "😀"},
		{name: "surrogate replaced", src: `k = {"\uD800"}` + "\n", want: "�"},
		{name: "above max replaced", src: `k = {"\UFFFFFF"}` + "\n", want: "�"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sl := firstStringLiteral(t, tc.src)
			assert.Equal(t, tc.want, sl.Parse())
		})
	}
}

// TestE0026AtEOF pins the E0026 (invalid unicode escape) annotation message to
// the reference: when \u runs out of hex digits at EOF, fluent.js interpolates
// ps.currentChar() which is `undefined`, yielding the literal "undefined" in the
// message (e.g. `\u00undefined`).
func TestE0026AtEOF(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		msg  string
	}{
		{
			name: "u escape truncated at EOF",
			src:  `k = {"\u00`,
			code: "E0026",
			msg:  `Invalid Unicode escape sequence: \u00undefined.`,
		},
		{
			name: "U escape truncated at EOF",
			src:  `k = {"\U0000`,
			code: "E0026",
			msg:  `Invalid Unicode escape sequence: \U0000undefined.`,
		},
		{
			// Non-EOF truncation: the offending char (the closing quote) is what
			// terminates the hex run; this must NOT print "undefined".
			name: "u escape truncated by quote",
			src:  `k = {"\u00"}` + "\n",
			code: "E0026",
			msg:  `Invalid Unicode escape sequence: \u00".`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := syntax.Parse(tc.src)

			var ann *ast.Annotation
			for _, e := range res.Body {
				if j, ok := e.(*ast.Junk); ok {
					for _, a := range j.Annotations {
						if a.Code == tc.code {
							ann = a
							break
						}
					}
				}
				if ann != nil {
					break
				}
			}
			require.NotNil(t, ann, "expected an E0026 annotation for %q", tc.src)
			assert.Equal(t, tc.code, ann.Code)
			assert.Equal(t, tc.msg, ann.Message)
		})
	}
}
