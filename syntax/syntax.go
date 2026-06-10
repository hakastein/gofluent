// Package syntax is an idiomatic Go port of @fluent/syntax (Project Fluent): a
// parser, serializer, and AST visitor for the Fluent localization format.
//
// The AST node types live in the sub-package
// github.com/hakastein/gofluent/syntax/ast.
package syntax

import (
	"unicode/utf16"

	"github.com/hakastein/gofluent/syntax/ast"
)

// Parse performs a full parse of source with junk recovery. It never returns an
// error: invalid entries become ast.Junk carrying ast.Annotation diagnostics.
// Spans are recorded by default; pass WithSpans(false) to disable them.
func Parse(source string, opts ...Option) *ast.Resource {
	return NewParser(opts...).Parse(source)
}

// ParseEntry parses the first Message or Term in source, skipping any preceding
// comments. It returns ast.Junk for unparseable input.
func ParseEntry(source string, opts ...Option) ast.Entry {
	return NewParser(opts...).ParseEntry(source)
}

// Serialize renders a resource to canonical Fluent source. Junk is omitted
// unless WithJunk(true) is supplied.
func Serialize(resource *ast.Resource, opts ...SerializerOption) string {
	return NewSerializer(opts...).Serialize(resource)
}

// LineOffset returns the zero-based line index of pos within source. pos is a
// UTF-16 code-unit offset, matching the offsets the parser records in spans
// (the parser is a port of fluent.js, whose stream indexes UTF-16 strings).
// This mirrors the reference lineOffset, which operates on JS (UTF-16) strings.
func LineOffset(source string, pos int) int {
	units := utf16.Encode([]rune(source))
	if pos > len(units) {
		pos = len(units)
	}
	if pos < 0 {
		pos = 0
	}
	count := 0
	for i := 0; i < pos; i++ {
		if units[i] == '\n' {
			count++
		}
	}
	return count
}

// ColumnOffset returns the zero-based column of pos within its line in source.
// As with LineOffset, pos is a UTF-16 code-unit offset and the scan is over
// UTF-16 units, mirroring the reference columnOffset (including its off-by-one:
// it searches backwards from pos-1 so a line break sitting exactly at pos is not
// treated as the previous one).
func ColumnOffset(source string, pos int) int {
	units := utf16.Encode([]rune(source))
	fromIndex := pos - 1
	if fromIndex > len(units)-1 {
		fromIndex = len(units) - 1
	}
	// lastIndexOf("\n", fromIndex): scan backwards from fromIndex (inclusive).
	prevLineBreak := -1
	for i := fromIndex; i >= 0; i-- {
		if units[i] == '\n' {
			prevLineBreak = i
			break
		}
	}
	if prevLineBreak == -1 {
		return pos
	}
	return pos - prevLineBreak - 1
}
