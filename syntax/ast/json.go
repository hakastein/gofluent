package ast

import (
	"bytes"
	"encoding/json"
)

// jsonField is a single key/value pair in an ordered JSON object.
type jsonField struct {
	key   string
	value any
}

// marshalCtx carries options through a marshaling pass.
type marshalCtx struct {
	withSpans bool
}

// jsonMarshaler is implemented by every AST node. It produces the ordered list
// of the node's own intrinsic fields; the "type" discriminator and the
// trailing "span" field are added by encodeObject.
type jsonMarshaler interface {
	Node
	jsonFields() []jsonField
}

// encodeObject writes an ordered JSON object for the given node, inserting the
// "type" discriminator first, then the node's intrinsic fields, then "span"
// last when spans are enabled and present.
func encodeObject(n jsonMarshaler, ctx *marshalCtx) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	buf.WriteString(`"type":`)
	tb, err := json.Marshal(n.nodeTypeName())
	if err != nil {
		return nil, err
	}
	buf.Write(tb)

	for _, f := range n.jsonFields() {
		buf.WriteByte(',')
		kb, err := json.Marshal(f.key)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := encodeValue(f.value, ctx)
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}

	// Span goes last. Span itself is a node but is encoded inline here, not as
	// part of jsonFields, because it is conditional and excluded from the Span
	// node's own field list. *Span is the one node that is not a SyntaxNode,
	// which also stops the recursion here.
	if sn, ok := n.(SyntaxNode); ok && ctx.withSpans {
		if sp := sn.GetSpan(); sp != nil {
			buf.WriteString(`,"span":`)
			sb, err := encodeObject(sp, ctx)
			if err != nil {
				return nil, err
			}
			buf.Write(sb)
		}
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// encodeValue encodes an arbitrary field value: nodes, slices of nodes, nil
// pointers, and scalars.
func encodeValue(v any, ctx *marshalCtx) ([]byte, error) {
	switch val := v.(type) {
	case nil:
		return []byte("null"), nil
	case jsonMarshaler:
		// Guard against typed-nil interfaces (e.g. (*Pattern)(nil)).
		if isNilNode(val) {
			return []byte("null"), nil
		}
		return encodeObject(val, ctx)
	default:
		return encodeSlice(v, ctx)
	}
}

// encodeSlice handles slices of node interfaces/pointers (and the []any of
// Annotation.Arguments), encoding each element as a value; a nil slice encodes
// as []. Non-slice scalars fall through to json.Marshal.
func encodeSlice(v any, ctx *marshalCtx) ([]byte, error) {
	switch s := v.(type) {
	case []any:
		return encodeNodeSlice(s, ctx)
	case []Entry:
		return encodeNodeSlice(s, ctx)
	case []PatternElement:
		return encodeNodeSlice(s, ctx)
	case []*Attribute:
		return encodeNodeSlice(s, ctx)
	case []*Variant:
		return encodeNodeSlice(s, ctx)
	case []InlineExpression:
		return encodeNodeSlice(s, ctx)
	case []*NamedArgument:
		return encodeNodeSlice(s, ctx)
	case []*Annotation:
		return encodeNodeSlice(s, ctx)
	default:
		return json.Marshal(v)
	}
}

func encodeNodeSlice[T any](items []T, ctx *marshalCtx) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, err := encodeValue(any(it), ctx)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}
