package ast

// Marshal encodes any AST node to canonical Fluent JSON matching the shape used
// by @fluent/syntax. When withSpans is false, every "span" field is omitted.
//
// The "type" discriminator equals the node's class name, field names and their
// order match the reference implementation, empty slices serialize as [] and
// absent optional nodes serialize as null.
func Marshal(n Node, withSpans bool) ([]byte, error) {
	return encodeValue(n, &marshalCtx{withSpans: withSpans})
}
