package fluent

import "sort"

// The runtime AST: the compact shapes produced by the runtime parser
// (resource.go) and consumed by the resolver — not the syntax AST. Only
// Pattern and Message are part of the public API; everything else is sealed
// inside the package.

// Pattern is a compiled message value or attribute. Obtain one from a
// Message (its Value field or an Attributes entry) and render it with
// Bundle.FormatPattern. Pattern is opaque: all implementations live in this
// package. A nil Pattern represents the absence of a value (e.g. a message
// with only attributes).
type Pattern interface {
	isPattern()
}

// textPattern is a simple pattern: plain text without placeables.
type textPattern string

func (textPattern) isPattern() {}

// complexPattern is a pattern with placeables.
type complexPattern []patternElement

func (complexPattern) isPattern() {}

// patternElement is an element of a complexPattern: a textElement or an
// expression.
type patternElement interface {
	isPatternElement()
}

// textElement is a verbatim text run within a complexPattern.
type textElement string

func (textElement) isPatternElement() {}

// expression is the union of all placeable expression types.
type expression interface {
	patternElement
	isExpression()
}

// literal is a stringLiteral or numberLiteral.
type literal interface {
	expression
	isLiteral()
}

// exprMarker makes the embedding struct an expression (and pattern element).
type exprMarker struct{}

func (exprMarker) isPatternElement() {}
func (exprMarker) isExpression()     {}

// litMarker additionally makes the embedding struct a literal.
type litMarker struct{ exprMarker }

func (litMarker) isLiteral() {}

// stringLiteral corresponds to ast.ts `StringLiteral` (type: "str").
type stringLiteral struct {
	litMarker
	value string
}

// numberLiteral corresponds to ast.ts `NumberLiteral` (type: "num").
type numberLiteral struct {
	litMarker
	value     float64
	precision int
}

// variableReference corresponds to ast.ts `VariableReference` (type: "var").
type variableReference struct {
	exprMarker
	name string
}

// messageReference corresponds to ast.ts `MessageReference` (type: "mesg").
type messageReference struct {
	exprMarker
	name string
	attr string // empty string means no attribute (ast.ts uses null)
}

// termReference corresponds to ast.ts `TermReference` (type: "term").
type termReference struct {
	exprMarker
	name string
	attr string // empty string means no attribute
	args callArguments
}

// functionReference corresponds to ast.ts `FunctionReference` (type: "func").
type functionReference struct {
	exprMarker
	name string
	args callArguments
}

// selectExpression corresponds to ast.ts `SelectExpression` (type: "select").
type selectExpression struct {
	exprMarker
	selector expression
	variants []variant
	star     int
}

// variant corresponds to ast.ts `Variant`.
type variant struct {
	key   literal
	value Pattern
}

// callArguments holds the arguments of a term or function call, split into
// positional and named at parse time.
type callArguments struct {
	positional []expression
	named      []namedArgument
}

// namedArgument corresponds to ast.ts `NamedArgument` (type: "narg").
type namedArgument struct {
	name  string
	value literal
}

// Message is a compiled message: its id, an optional value, and attributes.
// It is immutable — a Bundle shares one Message across every concurrent
// FormatPattern call — so its state is reached only through accessors, never
// mutable fields.
type Message struct {
	id         string
	value      Pattern // nil when the message has only attributes
	attributes map[string]Pattern
}

// ID returns the message identifier.
func (m *Message) ID() string { return m.id }

// Value returns the message's value pattern, or nil when the message has only
// attributes. Pass it to Bundle.FormatPattern to render it.
func (m *Message) Value() Pattern { return m.value }

// Attribute returns the pattern of the named attribute and whether it exists.
func (m *Message) Attribute(name string) (Pattern, bool) {
	p, ok := m.attributes[name]
	return p, ok
}

// AttributeNames returns the message's attribute names, sorted for
// deterministic iteration. The returned slice is a fresh copy.
func (m *Message) AttributeNames() []string {
	names := make([]string, 0, len(m.attributes))
	for name := range m.attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// term is the private counterpart of Message (its id starts with "-").
type term struct {
	id         string
	value      Pattern
	attributes map[string]Pattern
}

// entry is a top-level production in a Resource: *Message or *term.
type entry interface {
	isEntry()
}

func (*Message) isEntry() {}
func (*term) isEntry()    {}
