package fluent

import (
	"regexp"
	"strconv"
	"strings"
)

// This file ports fluent.js/fluent-bundle/src/resource.ts: the optimized
// runtime FTL parser producing the runtime AST (runtime_ast.go).
//
// The fluent.js parser uses sticky (/y) regexes anchored to a moving cursor.
// Go's regexp has no sticky flag, so each pattern is anchored with a leading
// `^` and matched against source[cursor:]; a match is only accepted when it
// begins at offset 0 of the slice. Capture-group indices are translated back to
// absolute offsets by adding the cursor.

var (
	reAttributeStart = regexp.MustCompile(`^\.([a-zA-Z][\w-]*) *= *`)
	reVariantStart   = regexp.MustCompile(`^\*?\[`)

	reNumberLiteral = regexp.MustCompile(`^(-?[0-9]+(?:\.([0-9]+))?)`)
	reIdentifier    = regexp.MustCompile(`^([a-zA-Z][\w-]*)`)
	reReference     = regexp.MustCompile(`^([$-])?([a-zA-Z][\w-]*)(?:\.([a-zA-Z][\w-]*))?`)
	reFunctionName  = regexp.MustCompile(`^[A-Z][A-Z0-9_-]*$`)

	reStringEscape  = regexp.MustCompile(`^\\([\\"])`)
	reUnicodeEscape = regexp.MustCompile(`^\\u([a-fA-F0-9]{4})|^\\U([a-fA-F0-9]{6})`)

	reLeadingNewlines = regexp.MustCompile(`^\n+`)
	reTrailingSpaces  = regexp.MustCompile(` +$`)
	reBlankLines      = regexp.MustCompile(` *\r?\n`)
	reIndent          = regexp.MustCompile(`( *)$`)

	tokenBraceOpen    = regexp.MustCompile(`^{(?: |\r?\n)*`)
	tokenBraceClose   = regexp.MustCompile(`^(?: |\r?\n)*}`)
	tokenBracketOpen  = regexp.MustCompile(`^\[(?: |\r?\n)*`)
	tokenBracketClose = regexp.MustCompile(`^(?: |\r?\n)*] *`)
	tokenParenOpen    = regexp.MustCompile(`^(?: |\r?\n)*\((?: |\r?\n)*`)
	tokenArrow        = regexp.MustCompile(`^(?: |\r?\n)*->(?: |\r?\n)*`)
	tokenColon        = regexp.MustCompile(`^(?: |\r?\n)*:(?: |\r?\n)*`)
	tokenComma        = regexp.MustCompile(`^(?: |\r?\n)*,?(?: |\r?\n)*`)
	tokenBlank        = regexp.MustCompile(`^(?: |\r?\n)+`)

	// reMessageScan finds the next message/term beginning at a line start.
	reMessageScan = regexp.MustCompile(`(?m)^(-?[a-zA-Z][\w-]*) *= *`)
)

// Resource is a structure storing parsed localization entries. Body holds
// *Message and *Term values, mirroring FluentResource.body in fluent.js.
type Resource struct {
	Body []any
}

// syntaxError is the bail-out signal for ill-formed messages, recovered per
// message in the parse loop (mirrors JS SyntaxError).
type syntaxError struct{ msg string }

func (e *syntaxError) Error() string { return e.msg }

// indent is the normalized blank block produced by makeIndent.
type indent struct {
	value  string
	length int
}

// parser holds the cursor-based parser state.
type parser struct {
	source string
	cursor int
}

// NewResource parses an FTL source into a Resource. Parse errors for individual
// messages are recovered (the message is skipped); the returned error slice is
// currently always nil-or-empty since the runtime parser silently skips broken
// entries, matching fluent.js. The signature returns it for API symmetry.
func NewResource(source string) (*Resource, []error) {
	res := &Resource{Body: []any{}}

	// Iterate over the beginnings of messages and terms to efficiently skip
	// comments and recover from errors. Emulate (?<!\r) by rejecting matches
	// whose preceding character is \r.
	locs := reMessageScan.FindAllStringSubmatchIndex(source, -1)
	for _, loc := range locs {
		idStart, idEnd := loc[2], loc[3]
		matchStart := loc[0]
		// (?<!\r): skip if the char right before the line start is \r. The (?m)
		// ^ matches after a \n; a \r\n line ending would place \r before it.
		if matchStart > 0 && source[matchStart-1] == '\r' {
			continue
		}
		id := source[idStart:idEnd]
		p := &parser{source: source, cursor: loc[1]}
		entry, err := p.parseMessage(id)
		if err != nil {
			// Don't report Fluent syntax errors; skip to the next message.
			if _, ok := err.(*syntaxError); ok {
				continue
			}
			return res, []error{err}
		}
		res.Body = append(res.Body, entry)
	}

	return res, nil
}

// --- low-level cursor primitives -------------------------------------------

func (p *parser) rest() string { return p.source[p.cursor:] }

// charAt returns the byte at the cursor offset n, or 0 if out of range.
func (p *parser) charAt(i int) byte {
	if i < 0 || i >= len(p.source) {
		return 0
	}
	return p.source[i]
}

// test reports whether re matches at the cursor.
func (p *parser) test(re *regexp.Regexp) bool {
	return re.MatchString(p.rest())
}

// consumeChar advances by one char if it matches. With mustErr, it returns a
// syntaxError when the char doesn't match.
func (p *parser) consumeChar(ch byte, mustErr bool) (bool, error) {
	if p.charAt(p.cursor) == ch {
		p.cursor++
		return true, nil
	}
	if mustErr {
		return false, &syntaxError{msg: "Expected " + string(ch)}
	}
	return false, nil
}

// consumeToken advances by the token if it matches. With mustErr it errors.
func (p *parser) consumeToken(re *regexp.Regexp, mustErr bool) (bool, error) {
	if m := re.FindStringIndex(p.rest()); m != nil {
		p.cursor += m[1]
		return true, nil
	}
	if mustErr {
		return false, &syntaxError{msg: "Expected token"}
	}
	return false, nil
}

// match runs re at the cursor, advances, and returns capture groups (as
// substrings; "" for groups that did not participate). It errors if no match.
func (p *parser) match(re *regexp.Regexp) ([]string, error) {
	rest := p.rest()
	idx := re.FindStringSubmatchIndex(rest)
	if idx == nil {
		return nil, &syntaxError{msg: "Expected match"}
	}
	groups := make([]string, len(idx)/2)
	for i := 0; i < len(idx)/2; i++ {
		s, e := idx[2*i], idx[2*i+1]
		if s < 0 {
			groups[i] = ""
		} else {
			groups[i] = rest[s:e]
		}
	}
	p.cursor += idx[1]
	return groups, nil
}

// match1 runs re and returns capture group 1.
func (p *parser) match1(re *regexp.Regexp) (string, error) {
	g, err := p.match(re)
	if err != nil {
		return "", err
	}
	return g[1], nil
}

// scanTextRun consumes a TextElement run: RE_TEXT_RUN =
// ((?:[^{}\n\r]|\r(?!\n))+). It does not match a \r immediately followed by \n.
// Returns (run, true) and advances if at least one char was consumed.
func (p *parser) scanTextRun() (string, bool) {
	start := p.cursor
	for p.cursor < len(p.source) {
		c := p.source[p.cursor]
		if c == '{' || c == '}' || c == '\n' {
			break
		}
		if c == '\r' {
			// Allowed only if not followed by \n.
			if p.cursor+1 < len(p.source) && p.source[p.cursor+1] == '\n' {
				break
			}
		}
		p.cursor++
	}
	if p.cursor == start {
		return "", false
	}
	return p.source[start:p.cursor], true
}

// scanStringRun consumes a StringLiteral run: RE_STRING_RUN =
// ((?:[^\\"\n\r]|\r(?!\n))*). May be empty.
func (p *parser) scanStringRun() string {
	start := p.cursor
	for p.cursor < len(p.source) {
		c := p.source[p.cursor]
		if c == '\\' || c == '"' || c == '\n' {
			break
		}
		if c == '\r' {
			if p.cursor+1 < len(p.source) && p.source[p.cursor+1] == '\n' {
				break
			}
		}
		p.cursor++
	}
	return p.source[start:p.cursor]
}

// --- grammar ---------------------------------------------------------------

func (p *parser) parseMessage(id string) (any, error) {
	value, err := p.parsePattern()
	if err != nil {
		return nil, err
	}
	attributes, err := p.parseAttributes()
	if err != nil {
		return nil, err
	}

	if value == nil && len(attributes) == 0 {
		return nil, &syntaxError{msg: "Expected message value or attributes"}
	}

	if strings.HasPrefix(id, "-") {
		return &Term{ID: id, Value: value, Attributes: attributes}, nil
	}
	return &Message{ID: id, Value: value, Attributes: attributes}, nil
}

func (p *parser) parseAttributes() (map[string]Pattern, error) {
	attrs := map[string]Pattern{}

	for p.testAttributeStart() {
		name, err := p.match1(reAttributeStart)
		if err != nil {
			return nil, err
		}
		value, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, &syntaxError{msg: "Expected attribute value"}
		}
		attrs[name] = value
	}

	return attrs, nil
}

// testAttributeStart emulates RE_ATTRIBUTE_START = /(?<=\n *)\.([a-zA-Z][\w-]*) *= */y
// The lookbehind requires that what precedes the cursor on this line is only
// spaces back to a newline (or start of source).
func (p *parser) testAttributeStart() bool {
	if p.charAt(p.cursor) != '.' {
		return false
	}
	// Verify lookbehind: walk backwards over spaces; must hit \n or start.
	i := p.cursor - 1
	for i >= 0 && p.source[i] == ' ' {
		i--
	}
	if i >= 0 && p.source[i] != '\n' {
		return false
	}
	return reAttributeStart.MatchString(p.rest())
}

func (p *parser) parsePattern() (Pattern, error) {
	var first string
	hasFirst := false

	if f, ok := p.scanTextRun(); ok {
		first = f
		hasFirst = true
	}

	// If there's a placeable on the first line, parse a complex pattern.
	if c := p.charAt(p.cursor); c == '{' || c == '}' {
		var elems []any
		if hasFirst {
			elems = []any{first}
		}
		return p.parsePatternElements(elems, 1<<30)
	}

	// Only continue if what comes after the newline is indented.
	ind, ok := p.parseIndent()
	if ok {
		if hasFirst {
			return p.parsePatternElements([]any{first, ind}, ind.length)
		}
		// Block pattern: discard leading newlines, keep inline indent.
		ind.value = reLeadingNewlines.ReplaceAllString(ind.value, "")
		return p.parsePatternElements([]any{ind}, ind.length)
	}

	if hasFirst {
		return reTrailingSpaces.ReplaceAllString(first, ""), nil
	}

	return nil, nil
}

// parsePatternElements parses a complex pattern as a slice of elements.
// elements may contain string and indent values.
func (p *parser) parsePatternElements(elements []any, commonIndent int) (ComplexPattern, error) {
	for {
		if t, ok := p.scanTextRun(); ok {
			elements = append(elements, t)
			continue
		}

		if p.charAt(p.cursor) == '{' {
			placeable, err := p.parsePlaceable()
			if err != nil {
				return nil, err
			}
			elements = append(elements, placeable)
			continue
		}

		if p.charAt(p.cursor) == '}' {
			return nil, &syntaxError{msg: "Unbalanced closing brace"}
		}

		ind, ok := p.parseIndent()
		if ok {
			elements = append(elements, ind)
			if ind.length < commonIndent {
				commonIndent = ind.length
			}
			continue
		}

		break
	}

	// Trim trailing spaces in the last element if it's a text string.
	if len(elements) > 0 {
		if last, ok := elements[len(elements)-1].(string); ok {
			elements[len(elements)-1] = reTrailingSpaces.ReplaceAllString(last, "")
		}
	}

	baked := ComplexPattern{}
	for _, element := range elements {
		if ind, ok := element.(indent); ok {
			// Dedent indented lines by the maximum common indent.
			s := ind.value
			if commonIndent <= len(s) {
				s = s[:len(s)-commonIndent]
			}
			element = s
		}
		if str, ok := element.(string); ok {
			if str == "" {
				continue
			}
			baked = append(baked, str)
			continue
		}
		// Non-string element (an Expression): always keep.
		baked = append(baked, element)
	}
	return baked, nil
}

func (p *parser) parsePlaceable() (Expression, error) {
	if _, err := p.consumeToken(tokenBraceOpen, true); err != nil {
		return nil, err
	}

	selector, err := p.parseInlineExpression()
	if err != nil {
		return nil, err
	}

	if ok, _ := p.consumeToken(tokenBraceClose, false); ok {
		return selector, nil
	}

	if ok, _ := p.consumeToken(tokenArrow, false); ok {
		variants, star, err := p.parseVariants()
		if err != nil {
			return nil, err
		}
		if _, err := p.consumeToken(tokenBraceClose, true); err != nil {
			return nil, err
		}
		return &SelectExpression{Selector: selector, Variants: variants, Star: star}, nil
	}

	return nil, &syntaxError{msg: "Unclosed placeable"}
}

func (p *parser) parseInlineExpression() (Expression, error) {
	if p.charAt(p.cursor) == '{' {
		return p.parsePlaceable()
	}

	if p.test(reReference) {
		g, err := p.match(reReference)
		if err != nil {
			return nil, err
		}
		sigil := g[1]
		name := g[2]
		attr := g[3] // "" if absent

		if sigil == "$" {
			return &VariableReference{Name: name}, nil
		}

		if ok, _ := p.consumeToken(tokenParenOpen, false); ok {
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}

			if sigil == "-" {
				return &TermReference{Name: name, Attr: attr, Args: args}, nil
			}

			if reFunctionName.MatchString(name) {
				return &FunctionReference{Name: name, Args: args}, nil
			}

			return nil, &syntaxError{msg: "Function names must be all upper-case"}
		}

		if sigil == "-" {
			return &TermReference{Name: name, Attr: attr, Args: []any{}}, nil
		}

		return &MessageReference{Name: name, Attr: attr}, nil
	}

	return p.parseLiteral()
}

func (p *parser) parseArguments() ([]any, error) {
	args := []any{}
	for {
		switch p.charAt(p.cursor) {
		case ')':
			p.cursor++
			return args, nil
		case 0:
			if p.cursor >= len(p.source) {
				return nil, &syntaxError{msg: "Unclosed argument list"}
			}
		}

		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		// Commas between arguments are treated as whitespace.
		p.consumeToken(tokenComma, false)
	}
}

func (p *parser) parseArgument() (any, error) {
	expr, err := p.parseInlineExpression()
	if err != nil {
		return nil, err
	}

	mesg, ok := expr.(*MessageReference)
	if !ok {
		return expr, nil
	}

	if ok, _ := p.consumeToken(tokenColon, false); ok {
		val, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &NamedArgument{Name: mesg.Name, Value: val}, nil
	}

	return expr, nil
}

func (p *parser) parseVariants() ([]Variant, int, error) {
	var variants []Variant
	count := 0
	star := -1

	for p.test(reVariantStart) {
		if ok, _ := p.consumeChar('*', false); ok {
			star = count
		}

		key, err := p.parseVariantKey()
		if err != nil {
			return nil, -1, err
		}
		value, err := p.parsePattern()
		if err != nil {
			return nil, -1, err
		}
		if value == nil {
			return nil, -1, &syntaxError{msg: "Expected variant value"}
		}
		variants = append(variants, Variant{Key: key, Value: value})
		count++
	}

	if count == 0 {
		return nil, -1, &syntaxError{msg: "Expected variants"}
	}

	if star == -1 {
		return nil, -1, &syntaxError{msg: "Expected default variant"}
	}

	return variants, star, nil
}

func (p *parser) parseVariantKey() (Literal, error) {
	if _, err := p.consumeToken(tokenBracketOpen, true); err != nil {
		return nil, err
	}

	var key Literal
	if p.test(reNumberLiteral) {
		k, err := p.parseNumberLiteral()
		if err != nil {
			return nil, err
		}
		key = k
	} else {
		id, err := p.match1(reIdentifier)
		if err != nil {
			return nil, err
		}
		key = &StringLiteral{Value: id}
	}

	if _, err := p.consumeToken(tokenBracketClose, true); err != nil {
		return nil, err
	}
	return key, nil
}

func (p *parser) parseLiteral() (Literal, error) {
	if p.test(reNumberLiteral) {
		return p.parseNumberLiteral()
	}

	if p.charAt(p.cursor) == '"' {
		return p.parseStringLiteral()
	}

	return nil, &syntaxError{msg: "Invalid expression"}
}

func (p *parser) parseNumberLiteral() (*NumberLiteral, error) {
	g, err := p.match(reNumberLiteral)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseFloat(g[1], 64)
	if err != nil {
		return nil, &syntaxError{msg: "Invalid number"}
	}
	precision := len(g[2])
	return &NumberLiteral{Value: value, Precision: precision}, nil
}

func (p *parser) parseStringLiteral() (*StringLiteral, error) {
	if _, err := p.consumeChar('"', true); err != nil {
		return nil, err
	}
	var sb strings.Builder
	for {
		sb.WriteString(p.scanStringRun())

		if p.charAt(p.cursor) == '\\' {
			esc, err := p.parseEscapeSequence()
			if err != nil {
				return nil, err
			}
			sb.WriteString(esc)
			continue
		}

		if ok, _ := p.consumeChar('"', false); ok {
			return &StringLiteral{Value: sb.String()}, nil
		}

		return nil, &syntaxError{msg: "Unclosed string literal"}
	}
}

func (p *parser) parseEscapeSequence() (string, error) {
	if p.test(reStringEscape) {
		return p.match1(reStringEscape)
	}

	if p.test(reUnicodeEscape) {
		g, err := p.match(reUnicodeEscape)
		if err != nil {
			return "", err
		}
		hex := g[1]
		if hex == "" {
			hex = g[2]
		}
		codepoint, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return "", &syntaxError{msg: "Invalid escape"}
		}
		// A \U escape can name up to 0xFFFFFF, but the largest valid Unicode
		// scalar is U+10FFFF. Reject anything above it explicitly (rather than
		// relying on Go's string(rune) silently yielding U+FFFD) so the intent is
		// clear.
		if codepoint > 0x10FFFF {
			return "�", nil
		}
		if codepoint <= 0xd7ff || codepoint >= 0xe000 {
			return string(rune(codepoint)), nil
		}
		// Lonely surrogate -> U+FFFD REPLACEMENT CHARACTER.
		return "�", nil
	}

	return "", &syntaxError{msg: "Unknown escape sequence"}
}

// parseIndent parses blank space. Returns the indent if it looks like indent
// before a pattern line; otherwise returns ok == false (and skips it).
func (p *parser) parseIndent() (indent, bool) {
	start := p.cursor
	p.consumeToken(tokenBlank, false)

	// EOF: end the pattern.
	if p.cursor >= len(p.source) {
		return indent{}, false
	}

	switch p.source[p.cursor] {
	case '.', '[', '*', '}':
		// A special character. End the pattern.
		return indent{}, false
	case '{':
		// Placeables don't require indentation. Continue the pattern.
		return p.makeIndent(p.source[start:p.cursor]), true
	}

	// Regular text char: require at least one space of indent before it.
	if p.cursor-1 >= 0 && p.source[p.cursor-1] == ' ' {
		return p.makeIndent(p.source[start:p.cursor]), true
	}

	return indent{}, false
}

// makeIndent normalizes a blank block and extracts the indent length.
func (p *parser) makeIndent(blank string) indent {
	value := reBlankLines.ReplaceAllString(blank, "\n")
	m := reIndent.FindStringSubmatch(blank)
	length := 0
	if m != nil {
		length = len(m[1])
	}
	return indent{value: value, length: length}
}
