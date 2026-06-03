// Package syntax is an idiomatic Go port of @fluent/syntax (Project Fluent): a
// parser, serializer, and AST visitor for the Fluent localization format.
//
// The AST node types live in the sub-package
// github.com/hakastein/gofluent/syntax/ast.
package syntax

import (
	"strings"

	"github.com/hakastein/gofluent/syntax/ast"
)

// Parse performs a full parse of source with junk recovery. It never returns an
// error: invalid entries become ast.Junk carrying ast.Annotation diagnostics.
// Spans are recorded by default; pass WithSpans(false) to disable them.
func Parse(source string, opts ...Option) *ast.Resource {
	return NewFluentParser(opts...).Parse(source)
}

// ParseEntry parses the first Message or Term in source, skipping any preceding
// comments. It returns ast.Junk (and a nil error) for unparseable input. The
// error return mirrors the reference parseEntry signature and is reserved for
// future use; it is always nil today.
func ParseEntry(source string, opts ...Option) (ast.Entry, error) {
	return NewFluentParser(opts...).ParseEntry(source)
}

// Serialize renders a resource to canonical Fluent source. Junk is omitted
// unless WithJunk(true) is supplied.
func Serialize(resource *ast.Resource, opts ...SerializerOption) string {
	return NewFluentSerializer(opts...).Serialize(resource)
}

// LineOffset returns the zero-based line index of pos within source.
func LineOffset(source string, pos int) int {
	if pos > len(source) {
		pos = len(source)
	}
	return strings.Count(source[:pos], "\n")
}

// ColumnOffset returns the zero-based column of pos within its line in source.
func ColumnOffset(source string, pos int) int {
	fromIndex := pos - 1
	if fromIndex > len(source) {
		fromIndex = len(source)
	}
	prevLineBreak := -1
	if fromIndex >= 0 {
		prevLineBreak = strings.LastIndex(source[:fromIndex+1], "\n")
	}
	if prevLineBreak == -1 {
		return pos
	}
	return pos - prevLineBreak - 1
}
