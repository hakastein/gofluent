// Package ast defines the Fluent abstract syntax tree, a faithful port of the
// data model from @fluent/syntax (ast.ts).
//
// Go has no class inheritance, so the abstract bases from the TypeScript
// implementation (SyntaxNode, Entry, Expression, Literal, BaseComment) are
// modeled as interfaces and the concrete productions as structs implementing
// them. Every node carries an optional *Span which is only populated when spans
// are enabled during parsing.
package ast

import (
	"reflect"
	"strconv"
	"strings"
)

// Node is the base interface implemented by every AST node, including Span and
// Annotation. It corresponds to BaseNode in the reference. Node is a closed
// interface: all implementations live in this package.
type Node interface {
	// nodeTypeName returns the node's class name used as the JSON "type"
	// discriminator.
	nodeTypeName() string
}

// SyntaxNode is the interface for nodes which carry their own Span via the
// Spanned embed. Every node in this package satisfies it except *Span, which
// implements only Node.
type SyntaxNode interface {
	Node
	GetSpan() *Span
	SetSpan(*Span)
	AddSpan(start, end int)
}

// Entry is one of the top-level productions in a Resource body: Message, Term,
// a comment, or Junk.
type Entry interface {
	SyntaxNode
	isEntry()
}

// Expression is any expression that may appear inside a Placeable.
type Expression interface {
	SyntaxNode
	isExpression()
}

// InlineExpression is the subset of expressions usable outside of a Placeable
// (everything except SelectExpression).
type InlineExpression interface {
	Expression
	isInlineExpression()
}

// PatternElement is an element of a Pattern: TextElement or Placeable.
type PatternElement interface {
	SyntaxNode
	isPatternElement()
}

// Literal is a StringLiteral or NumberLiteral.
type Literal interface {
	InlineExpression
	isLiteral()
}

// VariantKey is the key of a Variant: Identifier or NumberLiteral.
type VariantKey interface {
	SyntaxNode
	isVariantKey()
}

// BaseComment is a Comment, GroupComment, or ResourceComment.
type BaseComment interface {
	Entry
	isComment()
}

// Spanned is embedded by all SyntaxNode structs to provide span storage.
type Spanned struct {
	Span *Span
}

// GetSpan returns the node's span (nil when spans are disabled).
func (s *Spanned) GetSpan() *Span { return s.Span }

// SetSpan replaces the node's span.
func (s *Spanned) SetSpan(sp *Span) { s.Span = sp }

// AddSpan attaches a new span covering [start, end).
func (s *Spanned) AddSpan(start, end int) { s.Span = &Span{Start: start, End: end} }

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

// Resource is the root node holding a list of top-level entries.
type Resource struct {
	Spanned
	Body []Entry
}

func (*Resource) nodeTypeName() string { return "Resource" }

func (r *Resource) jsonFields() []jsonField {
	return []jsonField{{"body", r.Body}}
}

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

// Message is a translatable message with an optional value, attributes, and an
// optional attached comment.
type Message struct {
	Spanned
	ID         *Identifier
	Value      *Pattern // nil when the message has only attributes
	Attributes []*Attribute
	Comment    *Comment // nil when no comment is attached
}

func (*Message) nodeTypeName() string { return "Message" }
func (*Message) isEntry()             {}

func (m *Message) jsonFields() []jsonField {
	return []jsonField{
		{"id", m.ID},
		{"value", m.Value},
		{"attributes", m.Attributes},
		{"comment", m.Comment},
	}
}

// ---------------------------------------------------------------------------
// Term
// ---------------------------------------------------------------------------

// Term is a private message (its id is prefixed with "-"); it always has a
// value.
type Term struct {
	Spanned
	ID         *Identifier
	Value      *Pattern
	Attributes []*Attribute
	Comment    *Comment
}

func (*Term) nodeTypeName() string { return "Term" }
func (*Term) isEntry()             {}

func (t *Term) jsonFields() []jsonField {
	return []jsonField{
		{"id", t.ID},
		{"value", t.Value},
		{"attributes", t.Attributes},
		{"comment", t.Comment},
	}
}

// ---------------------------------------------------------------------------
// Pattern
// ---------------------------------------------------------------------------

// Pattern is the value of a message, term, attribute, or variant: a sequence of
// text elements and placeables.
type Pattern struct {
	Spanned
	Elements []PatternElement
}

func (*Pattern) nodeTypeName() string { return "Pattern" }

func (p *Pattern) jsonFields() []jsonField {
	return []jsonField{{"elements", p.Elements}}
}

// ---------------------------------------------------------------------------
// TextElement
// ---------------------------------------------------------------------------

// TextElement is verbatim text within a Pattern.
type TextElement struct {
	Spanned
	Value string
}

func (*TextElement) nodeTypeName() string { return "TextElement" }
func (*TextElement) isPatternElement()    {}

func (t *TextElement) jsonFields() []jsonField {
	return []jsonField{{"value", t.Value}}
}

// ---------------------------------------------------------------------------
// Placeable
// ---------------------------------------------------------------------------

// Placeable wraps an expression embedded in a Pattern (between { and }).
type Placeable struct {
	Spanned
	Expression Expression
}

func (*Placeable) nodeTypeName() string { return "Placeable" }
func (*Placeable) isPatternElement()    {}
func (*Placeable) isExpression()        {}
func (*Placeable) isInlineExpression()  {}

func (p *Placeable) jsonFields() []jsonField {
	return []jsonField{{"expression", p.Expression}}
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

// StringLiteral holds the exact, character-for-character contents of a quoted
// literal (escape sequences are not unescaped in the AST).
type StringLiteral struct {
	Spanned
	Value string
}

func (*StringLiteral) nodeTypeName() string { return "StringLiteral" }
func (*StringLiteral) isExpression()        {}
func (*StringLiteral) isInlineExpression()  {}
func (*StringLiteral) isLiteral()           {}

func (s *StringLiteral) jsonFields() []jsonField {
	return []jsonField{{"value", s.Value}}
}

// Parse unescapes the literal, returning the resolved string value. Escape
// sequences representing surrogate code points are replaced with U+FFFD.
func (s *StringLiteral) Parse() string {
	return unescapeStringLiteral(s.Value)
}

// NumberLiteral holds the textual form of a number literal.
type NumberLiteral struct {
	Spanned
	Value string
}

func (*NumberLiteral) nodeTypeName() string { return "NumberLiteral" }
func (*NumberLiteral) isExpression()        {}
func (*NumberLiteral) isInlineExpression()  {}
func (*NumberLiteral) isLiteral()           {}
func (*NumberLiteral) isVariantKey()        {}

func (n *NumberLiteral) jsonFields() []jsonField {
	return []jsonField{{"value", n.Value}}
}

// Parse returns the numeric value and the number of fractional digits.
func (n *NumberLiteral) Parse() (value float64, precision int) {
	value, _ = strconv.ParseFloat(n.Value, 64)
	if pos := strings.IndexByte(n.Value, '.'); pos > 0 {
		precision = len(n.Value) - pos - 1
	}
	return value, precision
}

// ---------------------------------------------------------------------------
// References
// ---------------------------------------------------------------------------

// MessageReference refers to another message, optionally to one of its
// attributes.
type MessageReference struct {
	Spanned
	ID        *Identifier
	Attribute *Identifier // nil when no attribute is referenced
}

func (*MessageReference) nodeTypeName() string { return "MessageReference" }
func (*MessageReference) isExpression()        {}
func (*MessageReference) isInlineExpression()  {}

func (m *MessageReference) jsonFields() []jsonField {
	return []jsonField{
		{"id", m.ID},
		{"attribute", m.Attribute},
	}
}

// TermReference refers to a term, optionally to one of its attributes, and may
// carry call arguments.
type TermReference struct {
	Spanned
	ID        *Identifier
	Attribute *Identifier    // nil when no attribute is referenced
	Arguments *CallArguments // nil when no arguments are present
}

func (*TermReference) nodeTypeName() string { return "TermReference" }
func (*TermReference) isExpression()        {}
func (*TermReference) isInlineExpression()  {}

func (t *TermReference) jsonFields() []jsonField {
	return []jsonField{
		{"id", t.ID},
		{"attribute", t.Attribute},
		{"arguments", t.Arguments},
	}
}

// VariableReference refers to an external variable ($name).
type VariableReference struct {
	Spanned
	ID *Identifier
}

func (*VariableReference) nodeTypeName() string { return "VariableReference" }
func (*VariableReference) isExpression()        {}
func (*VariableReference) isInlineExpression()  {}

func (v *VariableReference) jsonFields() []jsonField {
	return []jsonField{{"id", v.ID}}
}

// FunctionReference is a call to a built-in function (upper-case identifier).
type FunctionReference struct {
	Spanned
	ID        *Identifier
	Arguments *CallArguments
}

func (*FunctionReference) nodeTypeName() string { return "FunctionReference" }
func (*FunctionReference) isExpression()        {}
func (*FunctionReference) isInlineExpression()  {}

func (f *FunctionReference) jsonFields() []jsonField {
	return []jsonField{
		{"id", f.ID},
		{"arguments", f.Arguments},
	}
}

// ---------------------------------------------------------------------------
// SelectExpression
// ---------------------------------------------------------------------------

// SelectExpression chooses among variants based on a selector.
type SelectExpression struct {
	Spanned
	Selector InlineExpression
	Variants []*Variant
}

func (*SelectExpression) nodeTypeName() string { return "SelectExpression" }
func (*SelectExpression) isExpression()        {}

func (s *SelectExpression) jsonFields() []jsonField {
	return []jsonField{
		{"selector", s.Selector},
		{"variants", s.Variants},
	}
}

// ---------------------------------------------------------------------------
// CallArguments
// ---------------------------------------------------------------------------

// CallArguments holds the positional and named arguments of a call.
type CallArguments struct {
	Spanned
	Positional []InlineExpression
	Named      []*NamedArgument
}

func (*CallArguments) nodeTypeName() string { return "CallArguments" }

func (c *CallArguments) jsonFields() []jsonField {
	return []jsonField{
		{"positional", c.Positional},
		{"named", c.Named},
	}
}

// ---------------------------------------------------------------------------
// Attribute
// ---------------------------------------------------------------------------

// Attribute is a named sub-pattern of a Message or Term (.name = value).
type Attribute struct {
	Spanned
	ID    *Identifier
	Value *Pattern
}

func (*Attribute) nodeTypeName() string { return "Attribute" }

func (a *Attribute) jsonFields() []jsonField {
	return []jsonField{
		{"id", a.ID},
		{"value", a.Value},
	}
}

// ---------------------------------------------------------------------------
// Variant
// ---------------------------------------------------------------------------

// Variant is a single branch of a SelectExpression.
type Variant struct {
	Spanned
	Key     VariantKey // *Identifier or *NumberLiteral
	Value   *Pattern
	Default bool
}

func (*Variant) nodeTypeName() string { return "Variant" }

func (v *Variant) jsonFields() []jsonField {
	return []jsonField{
		{"key", v.Key},
		{"value", v.Value},
		{"default", v.Default},
	}
}

// ---------------------------------------------------------------------------
// NamedArgument
// ---------------------------------------------------------------------------

// NamedArgument is a key/value call argument (name: value).
type NamedArgument struct {
	Spanned
	Name  *Identifier
	Value Literal
}

func (*NamedArgument) nodeTypeName() string { return "NamedArgument" }

func (n *NamedArgument) jsonFields() []jsonField {
	return []jsonField{
		{"name", n.Name},
		{"value", n.Value},
	}
}

// ---------------------------------------------------------------------------
// Identifier
// ---------------------------------------------------------------------------

// Identifier is a bare name.
type Identifier struct {
	Spanned
	Name string
}

func (*Identifier) nodeTypeName() string { return "Identifier" }
func (*Identifier) isVariantKey()        {}

func (i *Identifier) jsonFields() []jsonField {
	return []jsonField{{"name", i.Name}}
}

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

// Comment is a standalone or attached comment (# ...).
type Comment struct {
	Spanned
	Content string
}

func (*Comment) nodeTypeName() string { return "Comment" }
func (*Comment) isEntry()             {}
func (*Comment) isComment()           {}

func (c *Comment) jsonFields() []jsonField {
	return []jsonField{{"content", c.Content}}
}

// GroupComment groups a section of a resource (## ...).
type GroupComment struct {
	Spanned
	Content string
}

func (*GroupComment) nodeTypeName() string { return "GroupComment" }
func (*GroupComment) isEntry()             {}
func (*GroupComment) isComment()           {}

func (c *GroupComment) jsonFields() []jsonField {
	return []jsonField{{"content", c.Content}}
}

// ResourceComment documents the whole resource (### ...).
type ResourceComment struct {
	Spanned
	Content string
}

func (*ResourceComment) nodeTypeName() string { return "ResourceComment" }
func (*ResourceComment) isEntry()             {}
func (*ResourceComment) isComment()           {}

func (c *ResourceComment) jsonFields() []jsonField {
	return []jsonField{{"content", c.Content}}
}

// ---------------------------------------------------------------------------
// Junk
// ---------------------------------------------------------------------------

// Junk is a slice of source that failed to parse, carrying the annotations that
// describe the errors.
type Junk struct {
	Spanned
	Annotations []*Annotation
	Content     string
}

func (*Junk) nodeTypeName() string { return "Junk" }
func (*Junk) isEntry()             {}

// AddAnnotation appends an annotation to the junk.
func (j *Junk) AddAnnotation(a *Annotation) {
	j.Annotations = append(j.Annotations, a)
}

func (j *Junk) jsonFields() []jsonField {
	return []jsonField{
		{"annotations", j.Annotations},
		{"content", j.Content},
	}
}

// ---------------------------------------------------------------------------
// Span
// ---------------------------------------------------------------------------

// Span records the [Start, End) range a node covers in the source, measured in
// UTF-16 code units (matching fluent.js); syntax.LineOffset and
// syntax.ColumnOffset convert such offsets to line/column positions.
type Span struct {
	Start int
	End   int
}

func (*Span) nodeTypeName() string { return "Span" }

func (s *Span) jsonFields() []jsonField {
	return []jsonField{
		{"start", s.Start},
		{"end", s.End},
	}
}

// ---------------------------------------------------------------------------
// Annotation
// ---------------------------------------------------------------------------

// Annotation describes a single parse error within Junk.
type Annotation struct {
	Spanned
	Code      string
	Arguments []any
	Message   string
}

func (*Annotation) nodeTypeName() string { return "Annotation" }

func (a *Annotation) jsonFields() []jsonField {
	return []jsonField{
		{"code", a.Code},
		{"arguments", a.Arguments},
		{"message", a.Message},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isNilNode reports whether an interface holds a typed-nil node pointer.
func isNilNode(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
