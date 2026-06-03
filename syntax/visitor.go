package syntax

import "github.com/hakastein/gofluent/syntax/ast"

// Visitor is a read-only AST visitor, a port of the Visitor base class from
// visitor.ts. Implementations receive each visited node via Visit. Returning
// true descends into the node's children via the generic walk; returning false
// stops descent for that subtree (the implementation is then responsible for
// any custom traversal, e.g. by calling Walk on selected children).
//
// This is the idiomatic Go shape of the reference's "add a visit{Type} method,
// then call genericVisit to descend" pattern: switch on the concrete node type
// inside Visit and return whether to auto-descend.
type Visitor interface {
	// Visit is called for a node. Return true to let Walk descend into the
	// node's children automatically.
	Visit(node ast.Node) (descend bool)
}

// Walk visits node with v, then (if v returned true) descends into each child
// node in declaration order. It corresponds to Visitor.visit + genericVisit.
func Walk(v Visitor, node ast.Node) {
	if node == nil || isNilNode(node) {
		return
	}
	if !v.Visit(node) {
		return
	}
	for _, child := range children(node) {
		Walk(v, child)
	}
}

// Transformer is a read-and-write visitor, a port of Transformer from
// visitor.ts. Transform is called for each node; the returned node replaces the
// original in the tree, and a nil return removes it. Return the node unchanged
// to keep it.
type Transformer interface {
	// Transform returns the replacement for node (possibly node itself), or nil
	// to remove it.
	Transform(node ast.Node) ast.Node
}

// Transform applies t to node and its descendants, rewriting children in place.
// It mirrors Transformer.visit + genericVisit: t is applied to node, and if the
// result is non-nil its children are recursively transformed.
func Transform(t Transformer, node ast.Node) ast.Node {
	if node == nil || isNilNode(node) {
		return nil
	}
	result := t.Transform(node)
	if result == nil || isNilNode(result) {
		return nil
	}
	transformChildren(t, result)
	return result
}

// isNilNode reports whether an ast.Node interface holds a typed-nil pointer.
func isNilNode(n ast.Node) bool {
	switch v := n.(type) {
	case *ast.Resource:
		return v == nil
	case *ast.Message:
		return v == nil
	case *ast.Term:
		return v == nil
	case *ast.Pattern:
		return v == nil
	case *ast.TextElement:
		return v == nil
	case *ast.Placeable:
		return v == nil
	case *ast.StringLiteral:
		return v == nil
	case *ast.NumberLiteral:
		return v == nil
	case *ast.MessageReference:
		return v == nil
	case *ast.TermReference:
		return v == nil
	case *ast.VariableReference:
		return v == nil
	case *ast.FunctionReference:
		return v == nil
	case *ast.SelectExpression:
		return v == nil
	case *ast.CallArguments:
		return v == nil
	case *ast.Attribute:
		return v == nil
	case *ast.Variant:
		return v == nil
	case *ast.NamedArgument:
		return v == nil
	case *ast.Identifier:
		return v == nil
	case *ast.Comment:
		return v == nil
	case *ast.GroupComment:
		return v == nil
	case *ast.ResourceComment:
		return v == nil
	case *ast.Junk:
		return v == nil
	case *ast.Span:
		return v == nil
	case *ast.Annotation:
		return v == nil
	case nil:
		return true
	default:
		return false
	}
}

// children returns the direct child nodes of node, in declaration order,
// skipping nil optionals. Span is not traversed (matching genericVisit, which
// only iterates own enumerable node-valued properties; span is a node but the
// reference visitor would visit it — however walking spans is rarely useful and
// the reference visitSpan hook is preserved by including it here).
func children(node ast.Node) []ast.Node {
	var out []ast.Node
	add := func(n ast.Node) {
		if n != nil && !isNilNode(n) {
			out = append(out, n)
		}
	}

	switch n := node.(type) {
	case *ast.Resource:
		for _, e := range n.Body {
			add(e)
		}
	case *ast.Message:
		add(n.ID)
		add(n.Value)
		for _, a := range n.Attributes {
			add(a)
		}
		add(n.Comment)
	case *ast.Term:
		add(n.ID)
		add(n.Value)
		for _, a := range n.Attributes {
			add(a)
		}
		add(n.Comment)
	case *ast.Pattern:
		for _, e := range n.Elements {
			add(e)
		}
	case *ast.Placeable:
		add(n.Expression)
	case *ast.MessageReference:
		add(n.ID)
		add(n.Attribute)
	case *ast.TermReference:
		add(n.ID)
		add(n.Attribute)
		add(n.Arguments)
	case *ast.VariableReference:
		add(n.ID)
	case *ast.FunctionReference:
		add(n.ID)
		add(n.Arguments)
	case *ast.SelectExpression:
		add(n.Selector)
		for _, v := range n.Variants {
			add(v)
		}
	case *ast.CallArguments:
		for _, e := range n.Positional {
			add(e)
		}
		for _, a := range n.Named {
			add(a)
		}
	case *ast.Attribute:
		add(n.ID)
		add(n.Value)
	case *ast.Variant:
		add(n.Key)
		add(n.Value)
	case *ast.NamedArgument:
		add(n.Name)
		add(n.Value)
	case *ast.Junk:
		for _, a := range n.Annotations {
			add(a)
		}
	}
	return out
}

// transformChildren applies t to each child slot of node, rewriting in place
// and dropping nil results from slices.
func transformChildren(t Transformer, node ast.Node) {
	switch n := node.(type) {
	case *ast.Resource:
		n.Body = transformEntries(t, n.Body)
	case *ast.Message:
		n.ID = transformIdent(t, n.ID)
		n.Value = transformPattern(t, n.Value)
		n.Attributes = transformAttrs(t, n.Attributes)
		n.Comment = transformCommentNode(t, n.Comment)
	case *ast.Term:
		n.ID = transformIdent(t, n.ID)
		n.Value = transformPattern(t, n.Value)
		n.Attributes = transformAttrs(t, n.Attributes)
		n.Comment = transformCommentNode(t, n.Comment)
	case *ast.Pattern:
		n.Elements = transformElements(t, n.Elements)
	case *ast.Placeable:
		if r := transformOne(t, n.Expression); r != nil {
			n.Expression, _ = r.(ast.Expression)
		} else {
			n.Expression = nil
		}
	case *ast.MessageReference:
		n.ID = transformIdent(t, n.ID)
		n.Attribute = transformIdent(t, n.Attribute)
	case *ast.TermReference:
		n.ID = transformIdent(t, n.ID)
		n.Attribute = transformIdent(t, n.Attribute)
		if n.Arguments != nil {
			if r := transformOne(t, n.Arguments); r != nil {
				n.Arguments, _ = r.(*ast.CallArguments)
			} else {
				n.Arguments = nil
			}
		}
	case *ast.VariableReference:
		n.ID = transformIdent(t, n.ID)
	case *ast.FunctionReference:
		n.ID = transformIdent(t, n.ID)
		if r := transformOne(t, n.Arguments); r != nil {
			n.Arguments, _ = r.(*ast.CallArguments)
		} else {
			n.Arguments = nil
		}
	case *ast.SelectExpression:
		if r := transformOne(t, n.Selector); r != nil {
			n.Selector, _ = r.(ast.InlineExpression)
		} else {
			n.Selector = nil
		}
		n.Variants = transformVariants(t, n.Variants)
	case *ast.CallArguments:
		n.Positional = transformInline(t, n.Positional)
		n.Named = transformNamed(t, n.Named)
	case *ast.Attribute:
		n.ID = transformIdent(t, n.ID)
		n.Value = transformPattern(t, n.Value)
	case *ast.Variant:
		if r := transformOne(t, n.Key); r != nil {
			n.Key, _ = r.(ast.VariantKey)
		} else {
			n.Key = nil
		}
		n.Value = transformPattern(t, n.Value)
	case *ast.NamedArgument:
		n.Name = transformIdent(t, n.Name)
		if r := transformOne(t, n.Value); r != nil {
			n.Value, _ = r.(ast.Literal)
		} else {
			n.Value = nil
		}
	case *ast.Junk:
		n.Annotations = transformAnnots(t, n.Annotations)
	}
}

func transformOne(t Transformer, n ast.Node) ast.Node {
	if n == nil || isNilNode(n) {
		return nil
	}
	return Transform(t, n)
}

func transformEntries(t Transformer, in []ast.Entry) []ast.Entry {
	out := make([]ast.Entry, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(ast.Entry); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformElements(t Transformer, in []ast.PatternElement) []ast.PatternElement {
	out := make([]ast.PatternElement, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(ast.PatternElement); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformAttrs(t Transformer, in []*ast.Attribute) []*ast.Attribute {
	out := make([]*ast.Attribute, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(*ast.Attribute); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformVariants(t Transformer, in []*ast.Variant) []*ast.Variant {
	out := make([]*ast.Variant, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(*ast.Variant); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformInline(t Transformer, in []ast.InlineExpression) []ast.InlineExpression {
	out := make([]ast.InlineExpression, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(ast.InlineExpression); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformNamed(t Transformer, in []*ast.NamedArgument) []*ast.NamedArgument {
	out := make([]*ast.NamedArgument, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(*ast.NamedArgument); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformAnnots(t Transformer, in []*ast.Annotation) []*ast.Annotation {
	out := make([]*ast.Annotation, 0, len(in))
	for _, e := range in {
		if r := transformOne(t, e); r != nil {
			if v, ok := r.(*ast.Annotation); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func transformIdent(t Transformer, in *ast.Identifier) *ast.Identifier {
	if in == nil {
		return nil
	}
	if r := transformOne(t, in); r != nil {
		if v, ok := r.(*ast.Identifier); ok {
			return v
		}
	}
	return nil
}

func transformPattern(t Transformer, in *ast.Pattern) *ast.Pattern {
	if in == nil {
		return nil
	}
	if r := transformOne(t, in); r != nil {
		if v, ok := r.(*ast.Pattern); ok {
			return v
		}
	}
	return nil
}

func transformCommentNode(t Transformer, in *ast.Comment) *ast.Comment {
	if in == nil {
		return nil
	}
	if r := transformOne(t, in); r != nil {
		if v, ok := r.(*ast.Comment); ok {
			return v
		}
	}
	return nil
}
