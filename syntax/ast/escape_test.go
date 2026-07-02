package ast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hakastein/gofluent/syntax/ast"
)

// TestStringLiteralParseUnescape pins the unescaper through the public
// StringLiteral.Parse API, with an emphasis on the character immediately
// following a \u/\U escape being preserved (the escape must consume exactly its
// own runes and nothing more). Each raw value is written with Go's "\\" so the
// stored StringLiteral.Value carries a single Fluent backslash.
func TestStringLiteralParseUnescape(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "digits follow bmp escape", raw: "\\u004100", want: "A00"},
		{name: "lone surrogate pair each replaced", raw: "\\uD83D\\uDE02", want: "��"},
		{name: "astral U escape", raw: "\\U01F602", want: "\U0001F602"},
		{name: "bmp escape at end of literal", raw: "A\\u0042", want: "AB"},
		{name: "consecutive escapes", raw: "\\u0041\\u0042", want: "AB"},
		{name: "escaped backslash after escape", raw: "\\u0041\\\\", want: "A\\"},
		{name: "escaped quote after escape", raw: "\\u0041\\\"", want: "A\""},
		{name: "escaped backslash before escape", raw: "\\\\\\u0041", want: "\\A"},
		{name: "out of range then literal", raw: "\\U110000!", want: "�!"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sl := &ast.StringLiteral{Value: tc.raw}
			assert.Equal(t, tc.want, sl.Parse())
		})
	}
}
