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
// of fields for the node, honoring the context options (e.g. whether spans are
// included). The "type" discriminator and the trailing "span" field are added
// by encodeObject; nodes only return their own intrinsic fields.
type jsonMarshaler interface {
	jsonFields(ctx *marshalCtx) []jsonField
}

// encodeNode encodes any AST node (or nil) into JSON bytes using the supplied
// context. nil values are encoded as JSON null.
func encodeNode(v any, ctx *marshalCtx) ([]byte, error) {
	switch n := v.(type) {
	case nil:
		return []byte("null"), nil
	case jsonMarshaler:
		return encodeObject(n, ctx)
	default:
		// Scalars, slices of scalars, etc.
		return json.Marshal(v)
	}
}

// encodeObject writes an ordered JSON object for the given node, inserting the
// "type" discriminator first, then the node's intrinsic fields, then "span"
// last when spans are enabled and present.
func encodeObject(n jsonMarshaler, ctx *marshalCtx) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	typeName := nodeType(n)
	buf.WriteString(`"type":`)
	tb, err := json.Marshal(typeName)
	if err != nil {
		return nil, err
	}
	buf.Write(tb)

	for _, f := range n.jsonFields(ctx) {
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
	// node's own field list.
	if sp := nodeSpan(n); sp != nil && ctx.withSpans {
		buf.WriteString(`,"span":`)
		sb, err := encodeObject(sp, ctx)
		if err != nil {
			return nil, err
		}
		buf.Write(sb)
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

// encodeSlice handles slices of node interfaces/pointers, encoding each element
// as a node. Non-node slices and scalars fall through to json.Marshal.
func encodeSlice(v any, ctx *marshalCtx) ([]byte, error) {
	switch s := v.(type) {
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
