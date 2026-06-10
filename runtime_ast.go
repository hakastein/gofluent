package fluent

// The runtime AST: the compact shapes produced by the runtime parser
// (resource.go) and consumed by the resolver — not the syntax AST.

// Pattern is either a simple string or a complex pattern (slice of elements).
// The concrete value is either a string or a ComplexPattern. A nil Pattern
// represents the absence of a value (e.g. a message with only attributes).
type Pattern = any

// ComplexPattern is an array of pattern elements.
type ComplexPattern []PatternElement

// PatternElement is either a string (text run) or an Expression (placeable).
// Concrete type is either `string` or one of the Expression structs below.
type PatternElement = any

// Expression is the union of all placeable expression types. Concrete type is
// one of: *SelectExpression, *VariableReference, *TermReference,
// *MessageReference, *FunctionReference, *StringLiteral, *NumberLiteral.
type Expression = any

// SelectExpression corresponds to ast.ts `SelectExpression` (type: "select").
type SelectExpression struct {
	Selector Expression
	Variants []Variant
	Star     int
}

// VariableReference corresponds to ast.ts `VariableReference` (type: "var").
type VariableReference struct {
	Name string
}

// TermReference corresponds to ast.ts `TermReference` (type: "term").
type TermReference struct {
	Name string
	Attr string // empty string means no attribute (ast.ts uses null)
	Args []any  // each element is an Expression or *NamedArgument
}

// MessageReference corresponds to ast.ts `MessageReference` (type: "mesg").
type MessageReference struct {
	Name string
	Attr string // empty string means no attribute
}

// FunctionReference corresponds to ast.ts `FunctionReference` (type: "func").
type FunctionReference struct {
	Name string
	Args []any // each element is an Expression or *NamedArgument
}

// Variant corresponds to ast.ts `Variant`.
type Variant struct {
	Key   Literal // either *StringLiteral or *NumberLiteral
	Value Pattern
}

// NamedArgument corresponds to ast.ts `NamedArgument` (type: "narg").
type NamedArgument struct {
	Name  string
	Value Literal // either *StringLiteral or *NumberLiteral
}

// Literal is the union of StringLiteral and NumberLiteral. Concrete type is
// either *StringLiteral or *NumberLiteral.
type Literal = any

// StringLiteral corresponds to ast.ts `StringLiteral` (type: "str").
type StringLiteral struct {
	Value string
}

// NumberLiteral corresponds to ast.ts `NumberLiteral` (type: "num").
type NumberLiteral struct {
	Value     float64
	Precision int
}

// Message is the raw runtime message shape `{id, value, attributes}`.
type Message struct {
	ID         string
	Value      Pattern // nil if the message has no value
	Attributes map[string]Pattern
}

// Term is the raw runtime term shape `{id, value, attributes}`.
type Term struct {
	ID         string
	Value      Pattern
	Attributes map[string]Pattern
}
