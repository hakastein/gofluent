package syntax

import (
	"strings"

	"github.com/hakastein/gofluent/syntax/ast"
)

// hasEntries is the serializer state bit indicating at least one entry has been
// emitted.
const hasEntries = 1

// FluentSerializer renders an AST back to canonical Fluent source. It is a port
// of FluentSerializer from serializer.ts.
type FluentSerializer struct {
	withJunk bool
}

// SerializerOption configures a FluentSerializer.
type SerializerOption func(*FluentSerializer)

// WithJunk includes Junk entries in the serialized output.
func WithJunk(enabled bool) SerializerOption {
	return func(s *FluentSerializer) { s.withJunk = enabled }
}

// NewFluentSerializer builds a serializer. Junk is omitted by default.
func NewFluentSerializer(opts ...SerializerOption) *FluentSerializer {
	s := &FluentSerializer{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Serialize renders a whole resource.
func (s *FluentSerializer) Serialize(resource *ast.Resource) string {
	state := 0
	var parts strings.Builder

	for _, entry := range resource.Body {
		if _, isJunk := entry.(*ast.Junk); !isJunk || s.withJunk {
			parts.WriteString(s.serializeEntry(entry, state))
			state |= hasEntries
		}
	}

	return parts.String()
}

// SerializeEntry renders a single entry with the given state bits.
func (s *FluentSerializer) serializeEntry(entry ast.Entry, state int) string {
	switch e := entry.(type) {
	case *ast.Message:
		return serializeMessage(e)
	case *ast.Term:
		return serializeTerm(e)
	case *ast.Comment:
		if state&hasEntries != 0 {
			return "\n" + serializeComment(e.Content, "#") + "\n"
		}
		return serializeComment(e.Content, "#") + "\n"
	case *ast.GroupComment:
		if state&hasEntries != 0 {
			return "\n" + serializeComment(e.Content, "##") + "\n"
		}
		return serializeComment(e.Content, "##") + "\n"
	case *ast.ResourceComment:
		if state&hasEntries != 0 {
			return "\n" + serializeComment(e.Content, "###") + "\n"
		}
		return serializeComment(e.Content, "###") + "\n"
	case *ast.Junk:
		return serializeJunk(e)
	default:
		panic("unknown entry type")
	}
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

	if isMultiline {
		if len(pattern.Elements) > 0 {
			if first, ok := pattern.Elements[0].(*ast.TextElement); ok && len(first.Value) > 0 {
				switch first.Value[0] {
				case '[', '.', '*':
					return false
				}
			}
		}
		return true
	}

	return false
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

func serializeJunk(junk *ast.Junk) string {
	return junk.Content
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
		return "{ " + SerializeExpression(expr) + "}"
	default:
		return "{ " + SerializeExpression(placeable.Expression) + " }"
	}
}

// SerializeExpression renders a single expression to Fluent source.
func SerializeExpression(expr ast.Expression) string {
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
		out := SerializeExpression(e.Selector) + " ->"
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
	key := SerializeVariantKey(variant.Key)
	value := indentExceptFirstLine(serializePattern(variant.Value))

	if variant.Default {
		return "\n   *[" + key + "]" + value
	}
	return "\n    [" + key + "]" + value
}

func serializeCallArguments(expr *ast.CallArguments) string {
	posParts := make([]string, len(expr.Positional))
	for i, p := range expr.Positional {
		posParts[i] = SerializeExpression(p)
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
	return arg.Name.Name + ": " + SerializeExpression(arg.Value)
}

// SerializeVariantKey renders a variant key (Identifier or NumberLiteral).
func SerializeVariantKey(key ast.VariantKey) string {
	switch k := key.(type) {
	case *ast.Identifier:
		return k.Name
	case *ast.NumberLiteral:
		return k.Value
	default:
		panic("unknown variant key type")
	}
}
