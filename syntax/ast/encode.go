package ast

// Marshal encodes any AST node to canonical Fluent JSON matching the shape used
// by @fluent/syntax. When withSpans is false, every "span" field is omitted,
// which mirrors the fixtures_structure comparison and is what the conformance
// suite uses to compare span-free trees.
//
// The "type" discriminator equals the node's class name, field names and their
// order match the reference implementation, empty slices serialize as [] and
// absent optional nodes serialize as null.
func Marshal(n Node, withSpans bool) ([]byte, error) {
	ctx := &marshalCtx{withSpans: withSpans}
	return encodeNode(n, ctx)
}

// MarshalJSON implements json.Marshaler for Resource, including spans. Use
// Marshal(node, false) to omit spans.
func (r *Resource) MarshalJSON() ([]byte, error) { return Marshal(r, true) }
