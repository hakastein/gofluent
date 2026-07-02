package syntax

import (
	"regexp"
	"strings"

	"github.com/hakastein/gofluent/syntax/ast"
)

// maxExpressionDepth bounds inline-expression nesting. A Go stack overflow is a
// fatal, unrecoverable error (unlike fluent.js, where deep nesting throws a
// catchable RangeError), so the depth is capped deliberately; real translations
// never nest anywhere near this. Exceeding it fails through the normal error
// path, yielding Junk like any other syntax error.
const maxExpressionDepth = 100

var trailingWSRe = regexp.MustCompile(`[ \n\r]+$`)

var functionNameRe = regexp.MustCompile(`^[A-Z][A-Z0-9_-]*$`)

// Parser is a recursive-descent parser for Fluent, a faithful port of the
// @fluent/syntax FluentParser. Construct one with NewParser, or use the
// package-level Parse / ParseEntry helpers.
type Parser struct {
	withSpans bool
}

// Option configures a Parser.
type Option func(*Parser)

// WithSpans enables or disables span recording. The default is true, matching
// @fluent/syntax.
func WithSpans(enabled bool) Option {
	return func(p *Parser) { p.withSpans = enabled }
}

// NewParser builds a parser with the given options. Spans are enabled by
// default.
func NewParser(opts ...Option) *Parser {
	p := &Parser{withSpans: true}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Parser) addSpan(node ast.SyntaxNode, start, end int) {
	if p.withSpans {
		node.AddSpan(start, end)
	}
}

// Parse performs a full parse with junk recovery. It never returns an error;
// invalid entries become Junk carrying Annotations.
func (p *Parser) Parse(source string) *ast.Resource {
	ps := newParserStream(source)
	ps.skipBlankBlock()

	var entries []ast.Entry
	var lastComment *ast.Comment

	for ps.currentChar() != eof {
		entry := p.getEntryOrJunk(ps)
		blankLines := ps.skipBlankBlock()

		// Regular Comments may be attached to a following Message or Term, but
		// must stand alone when followed by Junk. Stash and resolve next pass.
		if c, ok := entry.(*ast.Comment); ok && len(blankLines) == 0 && ps.currentChar() != eof {
			lastComment = c
			continue
		}

		if lastComment != nil {
			switch e := entry.(type) {
			case *ast.Message:
				e.Comment = lastComment
				if p.withSpans {
					e.GetSpan().Start = lastComment.GetSpan().Start
				}
			case *ast.Term:
				e.Comment = lastComment
				if p.withSpans {
					e.GetSpan().Start = lastComment.GetSpan().Start
				}
			default:
				entries = append(entries, lastComment)
			}
			lastComment = nil
		}

		entries = append(entries, entry)
	}

	res := &ast.Resource{Body: entries}
	p.addSpan(res, 0, ps.index)
	return res
}

// ParseEntry parses the first Message or Term in source, skipping preceding
// comments. Unparseable input yields ast.Junk.
func (p *Parser) ParseEntry(source string) ast.Entry {
	ps := newParserStream(source)
	ps.skipBlankBlock()

	for ps.currentChar() == '#' {
		skipped := p.getEntryOrJunk(ps)
		if _, ok := skipped.(*ast.Junk); ok {
			// Don't skip Junk comments.
			return skipped
		}
		ps.skipBlankBlock()
	}

	return p.getEntryOrJunk(ps)
}

func (p *Parser) getEntryOrJunk(ps *parserStream) ast.Entry {
	entryStartPos := ps.index

	entry, err := p.getEntry(ps)
	if err == nil {
		if err = ps.expectLineEnd(); err == nil {
			return entry
		}
	}

	pe := err.(*ParseError)

	errorIndex := ps.index
	ps.skipToNextEntryStart(entryStartPos)
	nextEntryStart := ps.index
	if nextEntryStart < errorIndex {
		errorIndex = nextEntryStart
	}

	slice := ps.slice(entryStartPos, nextEntryStart)
	junk := &ast.Junk{Content: slice}
	p.addSpan(junk, entryStartPos, nextEntryStart)
	annot := &ast.Annotation{Code: pe.Code, Arguments: pe.Args, Message: pe.Message}
	annot.AddSpan(errorIndex, errorIndex)
	junk.AddAnnotation(annot)
	return junk
}

func (p *Parser) getEntry(ps *parserStream) (ast.Entry, error) {
	var entry ast.Entry
	var err error

	switch {
	case ps.currentChar() == '#':
		var c ast.BaseComment
		c, err = p.getComment(ps)
		entry = c
	case ps.currentChar() == '-':
		var t *ast.Term
		t, err = p.getTerm(ps)
		entry = t
	case ps.isIdentifierStart():
		var m *ast.Message
		m, err = p.getMessage(ps)
		entry = m
	default:
		return nil, newParseError("E0002")
	}

	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (p *Parser) getComment(ps *parserStream) (ast.BaseComment, error) {
	start := ps.index
	// 0 - comment, 1 - group comment, 2 - resource comment
	level := -1
	var content []uint16

	for {
		i := -1
		limit := level
		if level == -1 {
			limit = 2
		}
		for ps.currentChar() == '#' && i < limit {
			ps.next()
			i++
		}

		if level == -1 {
			level = i
		}

		if ps.currentChar() != eol {
			if err := ps.expectChar(' '); err != nil {
				return nil, err
			}
			for {
				ch, ok := ps.takeChar(func(x rune) bool { return x != eol })
				if !ok {
					break
				}
				// ch may be a lone surrogate half of an astral character.
				content = appendRuneUTF16(content, ch)
			}
		}

		if ps.isNextLineComment(level) {
			content = appendRuneUTF16(content, ps.currentChar())
			ps.next()
		} else {
			break
		}
	}

	var comment ast.BaseComment
	text := decodeUTF16(content)
	switch level {
	case 0:
		comment = &ast.Comment{Content: text}
	case 1:
		comment = &ast.GroupComment{Content: text}
	default:
		comment = &ast.ResourceComment{Content: text}
	}
	p.addSpan(comment, start, ps.index)
	return comment, nil
}

func (p *Parser) getMessage(ps *parserStream) (*ast.Message, error) {
	start := ps.index
	id, err := p.getIdentifier(ps)
	if err != nil {
		return nil, err
	}

	ps.skipBlankInline()
	if err := ps.expectChar('='); err != nil {
		return nil, err
	}

	value, err := p.maybeGetPattern(ps)
	if err != nil {
		return nil, err
	}
	attrs, err := p.getAttributes(ps)
	if err != nil {
		return nil, err
	}

	if value == nil && len(attrs) == 0 {
		return nil, newParseError("E0005", id.Name)
	}

	msg := &ast.Message{ID: id, Value: value, Attributes: attrs}
	p.addSpan(msg, start, ps.index)
	return msg, nil
}

func (p *Parser) getTerm(ps *parserStream) (*ast.Term, error) {
	start := ps.index
	if err := ps.expectChar('-'); err != nil {
		return nil, err
	}
	id, err := p.getIdentifier(ps)
	if err != nil {
		return nil, err
	}

	ps.skipBlankInline()
	if err := ps.expectChar('='); err != nil {
		return nil, err
	}

	value, err := p.maybeGetPattern(ps)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newParseError("E0006", id.Name)
	}

	attrs, err := p.getAttributes(ps)
	if err != nil {
		return nil, err
	}

	term := &ast.Term{ID: id, Value: value, Attributes: attrs}
	p.addSpan(term, start, ps.index)
	return term, nil
}

func (p *Parser) getAttribute(ps *parserStream) (*ast.Attribute, error) {
	start := ps.index
	if err := ps.expectChar('.'); err != nil {
		return nil, err
	}

	key, err := p.getIdentifier(ps)
	if err != nil {
		return nil, err
	}

	ps.skipBlankInline()
	if err := ps.expectChar('='); err != nil {
		return nil, err
	}

	value, err := p.maybeGetPattern(ps)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newParseError("E0012")
	}

	attr := &ast.Attribute{ID: key, Value: value}
	p.addSpan(attr, start, ps.index)
	return attr, nil
}

func (p *Parser) getAttributes(ps *parserStream) ([]*ast.Attribute, error) {
	var attrs []*ast.Attribute
	ps.peekBlank()
	for ps.isAttributeStart() {
		ps.skipToPeek()
		attr, err := p.getAttribute(ps)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, attr)
		ps.peekBlank()
	}
	return attrs, nil
}

func (p *Parser) getIdentifier(ps *parserStream) (*ast.Identifier, error) {
	start := ps.index
	first, err := ps.takeIDStart()
	if err != nil {
		return nil, err
	}
	var name strings.Builder
	name.WriteRune(first)
	for {
		ch, ok := ps.takeIDChar()
		if !ok {
			break
		}
		name.WriteRune(ch)
	}
	id := &ast.Identifier{Name: name.String()}
	p.addSpan(id, start, ps.index)
	return id, nil
}

func (p *Parser) getVariantKey(ps *parserStream) (ast.VariantKey, error) {
	ch := ps.currentChar()
	if ch == eof {
		return nil, newParseError("E0013")
	}
	if (ch >= '0' && ch <= '9') || ch == '-' {
		return p.getNumber(ps)
	}
	return p.getIdentifier(ps)
}

func (p *Parser) getVariant(ps *parserStream, hasDefault bool) (*ast.Variant, error) {
	start := ps.index
	defaultIndex := false

	if ps.currentChar() == '*' {
		if hasDefault {
			return nil, newParseError("E0015")
		}
		ps.next()
		defaultIndex = true
	}

	if err := ps.expectChar('['); err != nil {
		return nil, err
	}

	ps.skipBlank()

	key, err := p.getVariantKey(ps)
	if err != nil {
		return nil, err
	}

	ps.skipBlank()
	if err := ps.expectChar(']'); err != nil {
		return nil, err
	}

	value, err := p.maybeGetPattern(ps)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newParseError("E0012")
	}

	variant := &ast.Variant{Key: key, Value: value, Default: defaultIndex}
	p.addSpan(variant, start, ps.index)
	return variant, nil
}

func (p *Parser) getVariants(ps *parserStream) ([]*ast.Variant, error) {
	var variants []*ast.Variant
	hasDefault := false

	ps.skipBlank()
	for ps.isVariantStart() {
		variant, err := p.getVariant(ps, hasDefault)
		if err != nil {
			return nil, err
		}
		if variant.Default {
			hasDefault = true
		}
		variants = append(variants, variant)
		if err := ps.expectLineEnd(); err != nil {
			return nil, err
		}
		ps.skipBlank()
	}

	if len(variants) == 0 {
		return nil, newParseError("E0011")
	}
	if !hasDefault {
		return nil, newParseError("E0010")
	}
	return variants, nil
}

func (p *Parser) getDigits(ps *parserStream) (string, error) {
	var num strings.Builder
	for {
		ch, ok := ps.takeDigit()
		if !ok {
			break
		}
		num.WriteRune(ch)
	}
	if num.Len() == 0 {
		return "", newParseError("E0004", "0-9")
	}
	return num.String(), nil
}

func (p *Parser) getNumber(ps *parserStream) (*ast.NumberLiteral, error) {
	start := ps.index
	var value strings.Builder

	if ps.currentChar() == '-' {
		ps.next()
		value.WriteByte('-')
	}
	digits, err := p.getDigits(ps)
	if err != nil {
		return nil, err
	}
	value.WriteString(digits)

	if ps.currentChar() == '.' {
		ps.next()
		digits, err := p.getDigits(ps)
		if err != nil {
			return nil, err
		}
		value.WriteByte('.')
		value.WriteString(digits)
	}

	num := &ast.NumberLiteral{Value: value.String()}
	p.addSpan(num, start, ps.index)
	return num, nil
}

// maybeGetPattern distinguishes patterns starting on the identifier line from
// block patterns starting on a new line; the distinction drives dedentation.
func (p *Parser) maybeGetPattern(ps *parserStream) (*ast.Pattern, error) {
	ps.peekBlankInline()
	if ps.isValueStart() {
		ps.skipToPeek()
		return p.getPattern(ps, false)
	}

	ps.peekBlankBlock()
	if ps.isValueContinuation() {
		ps.skipToPeek()
		return p.getPattern(ps, true)
	}

	return nil, nil
}

// indent is a transient token used while building patterns; it is not part of
// the AST.
type indent struct {
	value string
	span  *ast.Span
}

func (p *Parser) getPattern(ps *parserStream, isBlock bool) (*ast.Pattern, error) {
	start := ps.index
	// elements holds *ast.TextElement, *ast.Placeable, or *indent.
	var elements []any
	const inf = int(^uint(0) >> 1)
	commonIndentLength := inf

	if isBlock {
		blankStart := ps.index
		firstIndent := ps.skipBlankInline()
		elements = append(elements, p.getIndent(ps, firstIndent, blankStart))
		// Indent text is ASCII (spaces/newlines), so byte length equals the
		// UTF-16 code-unit length used by the reference.
		commonIndentLength = len(firstIndent)
	}

	for ps.currentChar() != eof {
		ch := ps.currentChar()
		switch ch {
		case eol:
			blankStart := ps.index
			blankLines := ps.peekBlankBlock()
			if ps.isValueContinuation() {
				ps.skipToPeek()
				ind := ps.skipBlankInline()
				if l := len(ind); l < commonIndentLength {
					commonIndentLength = l
				}
				elements = append(elements, p.getIndent(ps, blankLines+ind, blankStart))
				continue
			}
			ps.resetPeek(0)
			goto done
		case '{':
			pl, err := p.getPlaceable(ps)
			if err != nil {
				return nil, err
			}
			elements = append(elements, pl)
			continue
		case '}':
			return nil, newParseError("E0027")
		default:
			te, err := p.getTextElement(ps)
			if err != nil {
				return nil, err
			}
			elements = append(elements, te)
		}
	}

done:
	dedented := p.dedent(elements, commonIndentLength)
	pat := &ast.Pattern{Elements: dedented}
	p.addSpan(pat, start, ps.index)
	return pat, nil
}

func (p *Parser) getIndent(ps *parserStream, value string, start int) *indent {
	return &indent{value: value, span: &ast.Span{Start: start, End: ps.index}}
}

// dedent strips the common indent from text lines and merges adjacent text
// elements, mirroring the reference FluentParser.dedent.
func (p *Parser) dedent(elements []any, commonIndent int) []ast.PatternElement {
	var trimmed []ast.PatternElement

	// Adjacent text segments (TextElements and dedented indents) merge into a
	// single TextElement. Segments accumulate in a strings.Builder and flush
	// once, so a run of N segments costs O(N) rather than O(N^2) from repeated
	// string concatenation.
	var run strings.Builder
	runActive := false
	runStart, runEnd := 0, 0
	flush := func() {
		if !runActive {
			return
		}
		te := &ast.TextElement{Value: run.String()}
		if p.withSpans {
			te.AddSpan(runStart, runEnd)
		}
		trimmed = append(trimmed, te)
		run.Reset()
		runActive = false
	}

	for _, element := range elements {
		if pl, ok := element.(*ast.Placeable); ok {
			flush()
			trimmed = append(trimmed, pl)
			continue
		}

		var val string
		var segStart, segEnd int
		switch e := element.(type) {
		case *indent:
			keep := len(e.value) - commonIndent
			if keep < 0 {
				keep = 0
			}
			e.value = e.value[:keep]
			if len(e.value) == 0 {
				continue
			}
			val = e.value
			segStart, segEnd = e.span.Start, e.span.End
		case *ast.TextElement:
			val = e.Value
			if sp := e.GetSpan(); sp != nil {
				segStart, segEnd = sp.Start, sp.End
			}
		}

		if !runActive {
			runActive = true
			runStart = segStart
		}
		run.WriteString(val)
		runEnd = segEnd
	}
	flush()

	// Trim trailing whitespace from the Pattern.
	if len(trimmed) > 0 {
		if last, ok := trimmed[len(trimmed)-1].(*ast.TextElement); ok {
			last.Value = trailingWSRe.ReplaceAllString(last.Value, "")
			if len(last.Value) == 0 {
				trimmed = trimmed[:len(trimmed)-1]
			}
		}
	}

	return trimmed
}

func (p *Parser) getTextElement(ps *parserStream) (*ast.TextElement, error) {
	start := ps.index

	for ps.currentChar() != eof {
		ch := ps.currentChar()
		if ch == '{' || ch == '}' || ch == eol {
			break
		}
		ps.next()
	}

	te := &ast.TextElement{Value: ps.slice(start, ps.index)}
	p.addSpan(te, start, ps.index)
	return te, nil
}

func (p *Parser) getEscapeSequence(ps *parserStream) (string, error) {
	next := ps.currentChar()
	switch next {
	case '\\', '"':
		ps.next()
		return "\\" + string(next), nil
	case 'u':
		return p.getUnicodeEscapeSequence(ps, next, 4)
	case 'U':
		return p.getUnicodeEscapeSequence(ps, next, 6)
	case eof:
		// Mirror fluent.js: ps.currentChar() is `undefined` at EOF and renders
		// as the literal string "undefined".
		return "", newParseError("E0025", "undefined")
	default:
		return "", newParseError("E0025", string(next))
	}
}

func (p *Parser) getUnicodeEscapeSequence(ps *parserStream, u rune, digits int) (string, error) {
	if err := ps.expectChar(u); err != nil {
		return "", err
	}

	var sequence strings.Builder
	for i := 0; i < digits; i++ {
		ch, ok := ps.takeHexDigit()
		if !ok {
			cur := ps.currentChar()
			// Mirror fluent.js: the message interpolates ps.currentChar(), which
			// is `undefined` at EOF and renders as the literal string
			// "undefined" (e.g. \u00undefined). A real terminating char is
			// printed as-is.
			curStr := "undefined"
			if cur != eof {
				curStr = string(cur)
			}
			return "", newParseError("E0026", "\\"+string(u)+sequence.String()+curStr)
		}
		sequence.WriteRune(ch)
	}

	return "\\" + string(u) + sequence.String(), nil
}

func (p *Parser) getPlaceable(ps *parserStream) (*ast.Placeable, error) {
	start := ps.index
	if err := ps.expectChar('{'); err != nil {
		return nil, err
	}
	ps.skipBlank()
	expression, err := p.getExpression(ps)
	if err != nil {
		return nil, err
	}
	if err := ps.expectChar('}'); err != nil {
		return nil, err
	}
	pl := &ast.Placeable{Expression: expression}
	p.addSpan(pl, start, ps.index)
	return pl, nil
}

// getExpression returns an Expression. The result may be a Placeable when the
// inline expression is itself a placeable that is not a select.
func (p *Parser) getExpression(ps *parserStream) (ast.Expression, error) {
	start := ps.index
	selector, err := p.getInlineExpression(ps)
	if err != nil {
		return nil, err
	}
	ps.skipBlank()

	if ps.currentChar() == '-' {
		if ps.peek() != '>' {
			ps.resetPeek(0)
			return selector, nil
		}

		// Validate the selector per the Fluent spec.
		switch sel := selector.(type) {
		case *ast.MessageReference:
			if sel.Attribute == nil {
				return nil, newParseError("E0016")
			}
			return nil, newParseError("E0018")
		case *ast.TermReference:
			if sel.Attribute == nil {
				return nil, newParseError("E0017")
			}
		case *ast.Placeable:
			return nil, newParseError("E0029")
		}

		ps.next()
		ps.next()

		ps.skipBlankInline()
		if err := ps.expectLineEnd(); err != nil {
			return nil, err
		}

		variants, err := p.getVariants(ps)
		if err != nil {
			return nil, err
		}
		se := &ast.SelectExpression{Selector: selector, Variants: variants}
		p.addSpan(se, start, ps.index)
		return se, nil
	}

	if sel, ok := selector.(*ast.TermReference); ok && sel.Attribute != nil {
		return nil, newParseError("E0019")
	}

	return selector, nil
}

// getInlineExpression returns an InlineExpression. A Placeable also satisfies
// InlineExpression in this model.
func (p *Parser) getInlineExpression(ps *parserStream) (ast.InlineExpression, error) {
	ps.depth++
	defer func() { ps.depth-- }()
	if ps.depth > maxExpressionDepth {
		return nil, newParseError("E0028")
	}

	start := ps.index

	if ps.currentChar() == '{' {
		return p.getPlaceable(ps)
	}

	if ps.isNumberStart() {
		return p.getNumber(ps)
	}

	if ps.currentChar() == '"' {
		return p.getString(ps)
	}

	if ps.currentChar() == '$' {
		ps.next()
		id, err := p.getIdentifier(ps)
		if err != nil {
			return nil, err
		}
		vr := &ast.VariableReference{ID: id}
		p.addSpan(vr, start, ps.index)
		return vr, nil
	}

	if ps.currentChar() == '-' {
		ps.next()
		id, err := p.getIdentifier(ps)
		if err != nil {
			return nil, err
		}

		var attr *ast.Identifier
		if ps.currentChar() == '.' {
			ps.next()
			attr, err = p.getIdentifier(ps)
			if err != nil {
				return nil, err
			}
		}

		var args *ast.CallArguments
		ps.peekBlank()
		if ps.currentPeek() == '(' {
			ps.skipToPeek()
			args, err = p.getCallArguments(ps)
			if err != nil {
				return nil, err
			}
		}

		tr := &ast.TermReference{ID: id, Attribute: attr, Arguments: args}
		p.addSpan(tr, start, ps.index)
		return tr, nil
	}

	if ps.isIdentifierStart() {
		id, err := p.getIdentifier(ps)
		if err != nil {
			return nil, err
		}
		ps.peekBlank()

		if ps.currentPeek() == '(' {
			if !functionNameRe.MatchString(id.Name) {
				return nil, newParseError("E0008")
			}
			ps.skipToPeek()
			args, err := p.getCallArguments(ps)
			if err != nil {
				return nil, err
			}
			fr := &ast.FunctionReference{ID: id, Arguments: args}
			p.addSpan(fr, start, ps.index)
			return fr, nil
		}

		var attr *ast.Identifier
		if ps.currentChar() == '.' {
			ps.next()
			attr, err = p.getIdentifier(ps)
			if err != nil {
				return nil, err
			}
		}

		mr := &ast.MessageReference{ID: id, Attribute: attr}
		p.addSpan(mr, start, ps.index)
		return mr, nil
	}

	return nil, newParseError("E0028")
}

// callArgumentResult holds either an inline expression or a named argument.
type callArgumentResult struct {
	inline ast.InlineExpression
	named  *ast.NamedArgument
}

func (p *Parser) getCallArgument(ps *parserStream) (callArgumentResult, error) {
	start := ps.index
	exp, err := p.getInlineExpression(ps)
	if err != nil {
		return callArgumentResult{}, err
	}

	ps.skipBlank()

	if ps.currentChar() != ':' {
		return callArgumentResult{inline: exp}, nil
	}

	if mr, ok := exp.(*ast.MessageReference); ok && mr.Attribute == nil {
		ps.next()
		ps.skipBlank()

		value, err := p.getLiteral(ps)
		if err != nil {
			return callArgumentResult{}, err
		}
		na := &ast.NamedArgument{Name: mr.ID, Value: value}
		p.addSpan(na, start, ps.index)
		return callArgumentResult{named: na}, nil
	}

	return callArgumentResult{}, newParseError("E0009")
}

func (p *Parser) getCallArguments(ps *parserStream) (*ast.CallArguments, error) {
	start := ps.index
	var positional []ast.InlineExpression
	var named []*ast.NamedArgument
	argumentNames := map[string]bool{}

	if err := ps.expectChar('('); err != nil {
		return nil, err
	}
	ps.skipBlank()

	for {
		if ps.currentChar() == ')' {
			break
		}

		arg, err := p.getCallArgument(ps)
		if err != nil {
			return nil, err
		}
		if arg.named != nil {
			if argumentNames[arg.named.Name.Name] {
				return nil, newParseError("E0022")
			}
			named = append(named, arg.named)
			argumentNames[arg.named.Name.Name] = true
		} else if len(argumentNames) > 0 {
			return nil, newParseError("E0021")
		} else {
			positional = append(positional, arg.inline)
		}

		ps.skipBlank()

		if ps.currentChar() == ',' {
			ps.next()
			ps.skipBlank()
			continue
		}
		break
	}

	if err := ps.expectChar(')'); err != nil {
		return nil, err
	}
	ca := &ast.CallArguments{Positional: positional, Named: named}
	p.addSpan(ca, start, ps.index)
	return ca, nil
}

func (p *Parser) getString(ps *parserStream) (*ast.StringLiteral, error) {
	start := ps.index
	if err := ps.expectChar('"'); err != nil {
		return nil, err
	}
	var value []uint16

	for {
		ch, ok := ps.takeChar(func(x rune) bool { return x != '"' && x != eol })
		if !ok {
			break
		}
		if ch == '\\' {
			seq, err := p.getEscapeSequence(ps)
			if err != nil {
				return nil, err
			}
			for _, r := range seq {
				value = appendRuneUTF16(value, r)
			}
		} else {
			// ch may be a lone surrogate half of an astral character.
			value = appendRuneUTF16(value, ch)
		}
	}

	if ps.currentChar() == eol {
		return nil, newParseError("E0020")
	}

	if err := ps.expectChar('"'); err != nil {
		return nil, err
	}

	sl := &ast.StringLiteral{Value: decodeUTF16(value)}
	p.addSpan(sl, start, ps.index)
	return sl, nil
}

func (p *Parser) getLiteral(ps *parserStream) (ast.Literal, error) {
	if ps.isNumberStart() {
		return p.getNumber(ps)
	}
	if ps.currentChar() == '"' {
		return p.getString(ps)
	}
	return nil, newParseError("E0014")
}
