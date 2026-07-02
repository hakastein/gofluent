package syntax

import (
	"reflect"

	"github.com/hakastein/gofluent/syntax/ast"
)

// Visitor is a read-only AST visitor, a port of the Visitor base class from
// visitor.ts. Implementations receive each visited node via Visit. Returning
// true descends into the node's children via the generic walk; returning false
// stops descent for that subtree (the implementation is then responsible for
// any custom traversal, e.g. by calling Walk on selected children).
type Visitor interface {
	// Visit is called for a node. Return true to let Walk descend into the
	// node's children automatically.
	Visit(node ast.Node) (descend bool)
}

// Walk visits node with v, then (if v returned true) descends into each child
// node in declaration order. It corresponds to Visitor.visit + genericVisit.
func Walk(v Visitor, node ast.Node) {
	if isNilNode(node) {
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
// visitor.ts. Transform is called for each node; the returned node replaces
// the original in the tree, and a nil return removes it. Return the node
// unchanged to keep it.
type Transformer interface {
	// Transform returns the replacement for node (possibly node itself), or nil
	// to remove it.
	Transform(node ast.Node) ast.Node
}

// Transform applies t to node and its descendants, rewriting children in
// place. It mirrors Transformer.visit + genericVisit. If t returns nil for the
// root node, Transform returns nil.
func Transform(t Transformer, node ast.Node) ast.Node {
	if isNilNode(node) {
		return nil
	}
	result := t.Transform(node)
	if isNilNode(result) {
		return nil
	}
	transformChildren(t, result)
	return result
}

// isNilNode reports whether node is nil or holds a typed-nil pointer.
func isNilNode(n ast.Node) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// children returns the direct child nodes of node, in declaration order,
// skipping nil optionals. The node's own Span is appended last, matching the
// reference genericVisit (which iterates the `span` property too).
func children(node ast.Node) []ast.Node {
	var out []ast.Node
	add := func(n ast.Node) {
		if !isNilNode(n) {
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

	if sn, ok := node.(ast.SyntaxNode); ok {
		if sp := sn.GetSpan(); sp != nil {
			out = append(out, sp)
		}
	}
	return out
}

// transformChildren applies t to each child slot of node, rewriting in place
// and dropping nil results from slices.
func transformChildren(t Transformer, node ast.Node) {
	switch n := node.(type) {
	case *ast.Resource:
		n.Body = transformSlice(t, n.Body)
	case *ast.Message:
		n.ID = transformNode(t, n.ID)
		n.Value = transformNode(t, n.Value)
		n.Attributes = transformSlice(t, n.Attributes)
		n.Comment = transformNode(t, n.Comment)
	case *ast.Term:
		n.ID = transformNode(t, n.ID)
		n.Value = transformNode(t, n.Value)
		n.Attributes = transformSlice(t, n.Attributes)
		n.Comment = transformNode(t, n.Comment)
	case *ast.Pattern:
		n.Elements = transformSlice(t, n.Elements)
	case *ast.Placeable:
		n.Expression = transformNode(t, n.Expression)
	case *ast.MessageReference:
		n.ID = transformNode(t, n.ID)
		n.Attribute = transformNode(t, n.Attribute)
	case *ast.TermReference:
		n.ID = transformNode(t, n.ID)
		n.Attribute = transformNode(t, n.Attribute)
		n.Arguments = transformNode(t, n.Arguments)
	case *ast.VariableReference:
		n.ID = transformNode(t, n.ID)
	case *ast.FunctionReference:
		n.ID = transformNode(t, n.ID)
		n.Arguments = transformNode(t, n.Arguments)
	case *ast.SelectExpression:
		n.Selector = transformNode(t, n.Selector)
		n.Variants = transformSlice(t, n.Variants)
	case *ast.CallArguments:
		n.Positional = transformSlice(t, n.Positional)
		n.Named = transformSlice(t, n.Named)
	case *ast.Attribute:
		n.ID = transformNode(t, n.ID)
		n.Value = transformNode(t, n.Value)
	case *ast.Variant:
		n.Key = transformNode(t, n.Key)
		n.Value = transformNode(t, n.Value)
	case *ast.NamedArgument:
		n.Name = transformNode(t, n.Name)
		n.Value = transformNode(t, n.Value)
	case *ast.Junk:
		n.Annotations = transformSlice(t, n.Annotations)
	}

	// The reference genericVisit also transforms the node's `span` property.
	if sn, ok := node.(ast.SyntaxNode); ok {
		if sp := sn.GetSpan(); sp != nil {
			sn.SetSpan(transformNode(t, sp))
		}
	}
}

// transformNode transforms a single child node, returning the zero value when
// the transformer removes it or replaces it with a node of a different type.
func transformNode[T ast.Node](t Transformer, n T) T {
	if v, ok := Transform(t, n).(T); ok {
		return v
	}
	var zero T
	return zero
}

// transformSlice transforms each element of in, dropping elements the
// transformer removes or replaces with a node of a different type.
func transformSlice[T ast.Node](t Transformer, in []T) []T {
	out := make([]T, 0, len(in))
	for _, e := range in {
		if v, ok := Transform(t, e).(T); ok {
			out = append(out, v)
		}
	}
	return out
}
