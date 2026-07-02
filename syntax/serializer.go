package syntax

import (
	"strings"

	"github.com/hakastein/gofluent/syntax/ast"
)

// Serializer renders an AST back to canonical Fluent source. It is a port of
// the @fluent/syntax FluentSerializer.
type Serializer struct {
	withJunk bool
}

// SerializerOption configures a Serializer.
type SerializerOption func(*Serializer)

// WithJunk includes Junk entries in the serialized output.
func WithJunk(enabled bool) SerializerOption {
	return func(s *Serializer) { s.withJunk = enabled }
}

// NewSerializer builds a serializer. Junk is omitted by default.
func NewSerializer(opts ...SerializerOption) *Serializer {
	s := &Serializer{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Serialize renders a whole resource.
func (s *Serializer) Serialize(resource *ast.Resource) string {
	hasEntries := false
	var parts strings.Builder

	for _, entry := range resource.Body {
		if _, isJunk := entry.(*ast.Junk); !isJunk || s.withJunk {
			parts.WriteString(s.serializeEntry(entry, hasEntries))
			hasEntries = true
		}
	}

	return parts.String()
}

func (s *Serializer) serializeEntry(entry ast.Entry, hasEntries bool) string {
	switch e := entry.(type) {
	case *ast.Message:
		return serializeMessage(e)
	case *ast.Term:
		return serializeTerm(e)
	case *ast.Comment:
		return serializeStandaloneComment(e.Content, "#", hasEntries)
	case *ast.GroupComment:
		return serializeStandaloneComment(e.Content, "##", hasEntries)
	case *ast.ResourceComment:
		return serializeStandaloneComment(e.Content, "###", hasEntries)
	case *ast.Junk:
		return e.Content
	default:
		panic("unknown entry type")
	}
}

// serializeStandaloneComment renders a standalone comment entry; every entry
// after the first is preceded by a separating blank line.
func serializeStandaloneComment(content, prefix string, hasEntries bool) string {
	out := serializeComment(content, prefix) + "\n"
	if hasEntries {
		return "\n" + out
	}
	return out
}

func indentExceptFirstLine(content string) string {
	return strings.Join(strings.Split(content, "\n"), "\n    ")
}

func includesNewLine(elem ast.PatternElement) bool {
	te, ok := elem.(*ast.TextElement)
	return ok && strings.Contains(te.Value, "\n")
}

func isSelectExpr(elem ast.PatternElement) bool {
	pl, ok := elem.(*ast.Placeable)
	if !ok {
		return false
	}
	_, ok = pl.Expression.(*ast.SelectExpression)
	return ok
}

func shouldStartOnNewLine(pattern *ast.Pattern) bool {
	isMultiline := false
	for _, e := range pattern.Elements {
		if isSelectExpr(e) || includesNewLine(e) {
			isMultiline = true
			break
		}
	}
	if !isMultiline {
		return false
	}

	// A leading [, . or * on the first line would be confused with a variant
	// key, an attribute, or the default variant marker.
	if first, ok := pattern.Elements[0].(*ast.TextElement); ok && len(first.Value) > 0 {
		switch first.Value[0] {
		case '[', '.', '*':
			return false
		}
	}
	return true
}

func serializeComment(content, prefix string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			lines[i] = prefix + " " + line
		} else {
			lines[i] = prefix
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func serializeMessage(message *ast.Message) string {
	var parts strings.Builder

	if message.Comment != nil {
		parts.WriteString(serializeComment(message.Comment.Content, "#"))
	}

	parts.WriteString(message.ID.Name)
	parts.WriteString(" =")

	if message.Value != nil {
		parts.WriteString(serializePattern(message.Value))
	}

	for _, attribute := range message.Attributes {
		parts.WriteString(serializeAttribute(attribute))
	}

	parts.WriteString("\n")
	return parts.String()
}

func serializeTerm(term *ast.Term) string {
	var parts strings.Builder

	if term.Comment != nil {
		parts.WriteString(serializeComment(term.Comment.Content, "#"))
	}

	parts.WriteString("-")
	parts.WriteString(term.ID.Name)
	parts.WriteString(" =")
	parts.WriteString(serializePattern(term.Value))

	for _, attribute := range term.Attributes {
		parts.WriteString(serializeAttribute(attribute))
	}

	parts.WriteString("\n")
	return parts.String()
}

func serializeAttribute(attribute *ast.Attribute) string {
	value := indentExceptFirstLine(serializePattern(attribute.Value))
	return "\n    ." + attribute.ID.Name + " =" + value
}

func serializePattern(pattern *ast.Pattern) string {
	var content strings.Builder
	for _, e := range pattern.Elements {
		content.WriteString(serializeElement(e))
	}
	c := content.String()

	if shouldStartOnNewLine(pattern) {
		return "\n    " + indentExceptFirstLine(c)
	}
	return " " + indentExceptFirstLine(c)
}

func serializeElement(element ast.PatternElement) string {
	switch e := element.(type) {
	case *ast.TextElement:
		return e.Value
	case *ast.Placeable:
		return serializePlaceable(e)
	default:
		panic("unknown element type")
	}
}

func serializePlaceable(placeable *ast.Placeable) string {
	switch expr := placeable.Expression.(type) {
	case *ast.Placeable:
		return "{" + serializePlaceable(expr) + "}"
	case *ast.SelectExpression:
		// Control the whitespace around the braces for select expressions.
		return "{ " + serializeExpression(expr) + "}"
	default:
		return "{ " + serializeExpression(placeable.Expression) + " }"
	}
}

func serializeExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return "\"" + e.Value + "\""
	case *ast.NumberLiteral:
		return e.Value
	case *ast.VariableReference:
		return "$" + e.ID.Name
	case *ast.TermReference:
		out := "-" + e.ID.Name
		if e.Attribute != nil {
			out += "." + e.Attribute.Name
		}
		if e.Arguments != nil {
			out += serializeCallArguments(e.Arguments)
		}
		return out
	case *ast.MessageReference:
		out := e.ID.Name
		if e.Attribute != nil {
			out += "." + e.Attribute.Name
		}
		return out
	case *ast.FunctionReference:
		return e.ID.Name + serializeCallArguments(e.Arguments)
	case *ast.SelectExpression:
		out := serializeExpression(e.Selector) + " ->"
		for _, variant := range e.Variants {
			out += serializeVariant(variant)
		}
		return out + "\n"
	case *ast.Placeable:
		return serializePlaceable(e)
	default:
		panic("unknown expression type")
	}
}

func serializeVariant(variant *ast.Variant) string {
	key := serializeVariantKey(variant.Key)
	value := indentExceptFirstLine(serializePattern(variant.Value))

	if variant.Default {
		return "\n   *[" + key + "]" + value
	}
	return "\n    [" + key + "]" + value
}

func serializeCallArguments(expr *ast.CallArguments) string {
	posParts := make([]string, len(expr.Positional))
	for i, p := range expr.Positional {
		posParts[i] = serializeExpression(p)
	}
	positional := strings.Join(posParts, ", ")

	namedParts := make([]string, len(expr.Named))
	for i, n := range expr.Named {
		namedParts[i] = serializeNamedArgument(n)
	}
	named := strings.Join(namedParts, ", ")

	if len(expr.Positional) > 0 && len(expr.Named) > 0 {
		return "(" + positional + ", " + named + ")"
	}
	if positional != "" {
		return "(" + positional + ")"
	}
	return "(" + named + ")"
}

func serializeNamedArgument(arg *ast.NamedArgument) string {
	return arg.Name.Name + ": " + serializeExpression(arg.Value)
}

func serializeVariantKey(key ast.VariantKey) string {
	switch k := key.(type) {
	case *ast.Identifier:
		return k.Name
	case *ast.NumberLiteral:
		return k.Value
	default:
		panic("unknown variant key type")
	}
}
