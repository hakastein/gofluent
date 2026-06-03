package syntax

import "unicode/utf16"

// eof is the sentinel returned by the stream when the cursor is past the end of
// the input. It mirrors the EOF === undefined check in the reference. We use a
// value that can never be a real UTF-16 code unit.
const eof rune = -1

// eol is the canonical end-of-line character. CRLF is normalized to LF.
const eol rune = '\n'

// specialLineStartChars are characters that cannot start a pattern continuation
// line.
var specialLineStartChars = map[rune]bool{'}': true, '.': true, '[': true, '*': true}

// parserStream is a cursor over the source, indexed by UTF-16 code units to
// match the offset semantics of @fluent/syntax (and therefore the fixture
// spans). It is a port of ParserStream/FluentParserStream from stream.ts.
type parserStream struct {
	units      []uint16
	index      int
	peekOffset int
}

func newParserStream(source string) *parserStream {
	return &parserStream{units: utf16.Encode([]rune(source))}
}

// unitAt returns the raw UTF-16 unit at i, or eof past the end.
func (ps *parserStream) unitAt(i int) rune {
	if i < 0 || i >= len(ps.units) {
		return eof
	}
	return rune(ps.units[i])
}

// charAt returns the logical character at offset, normalizing CRLF to LF
// without moving the cursor.
func (ps *parserStream) charAt(offset int) rune {
	if ps.unitAt(offset) == '\r' && ps.unitAt(offset+1) == '\n' {
		return '\n'
	}
	return ps.unitAt(offset)
}

func (ps *parserStream) currentChar() rune {
	return ps.charAt(ps.index)
}

func (ps *parserStream) currentPeek() rune {
	return ps.charAt(ps.index + ps.peekOffset)
}

// next advances the cursor by one logical character (skipping CRLF as one) and
// returns the new current raw unit.
func (ps *parserStream) next() rune {
	ps.peekOffset = 0
	if ps.unitAt(ps.index) == '\r' && ps.unitAt(ps.index+1) == '\n' {
		ps.index++
	}
	ps.index++
	return ps.unitAt(ps.index)
}

// peek advances the peek cursor by one logical character and returns the raw
// unit at the new peek position.
func (ps *parserStream) peek() rune {
	if ps.unitAt(ps.index+ps.peekOffset) == '\r' && ps.unitAt(ps.index+ps.peekOffset+1) == '\n' {
		ps.peekOffset++
	}
	ps.peekOffset++
	return ps.unitAt(ps.index + ps.peekOffset)
}

func (ps *parserStream) resetPeek(offset int) {
	ps.peekOffset = offset
}

func (ps *parserStream) skipToPeek() {
	ps.index += ps.peekOffset
	ps.peekOffset = 0
}

// slice reconstructs the source substring spanning [start, end) UTF-16 units.
func (ps *parserStream) slice(start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(ps.units) {
		end = len(ps.units)
	}
	if start >= end {
		return ""
	}
	return string(utf16.Decode(ps.units[start:end]))
}

// appendRuneUTF16 appends a rune to a UTF-16 unit buffer. A rune in the
// surrogate range (a lone half produced by reading a single UTF-16 unit) is
// stored verbatim so astral-plane characters survive reconstruction; other
// runes are re-encoded to UTF-16. This mirrors how the reference operates on
// UTF-16 strings unit by unit.
func appendRuneUTF16(buf []uint16, r rune) []uint16 {
	if r >= 0xD800 && r <= 0xDFFF {
		return append(buf, uint16(r))
	}
	return utf16.AppendRune(buf, r)
}

// decodeUTF16 turns a UTF-16 unit buffer back into a Go string.
func decodeUTF16(buf []uint16) string {
	return string(utf16.Decode(buf))
}

// lastIndexOfEOL returns the index of the last LF at or before `from`, or -1.
func (ps *parserStream) lastIndexOfEOL(from int) int {
	if from > len(ps.units) {
		from = len(ps.units)
	}
	for i := from; i >= 0; i-- {
		if ps.unitAt(i) == '\n' {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// FluentParserStream methods
// ---------------------------------------------------------------------------

func (ps *parserStream) peekBlankInline() string {
	start := ps.index + ps.peekOffset
	for ps.currentPeek() == ' ' {
		ps.peek()
	}
	return ps.slice(start, ps.index+ps.peekOffset)
}

func (ps *parserStream) skipBlankInline() string {
	blank := ps.peekBlankInline()
	ps.skipToPeek()
	return blank
}

func (ps *parserStream) peekBlankBlock() string {
	blank := ""
	for {
		lineStart := ps.peekOffset
		ps.peekBlankInline()
		if ps.currentPeek() == eol {
			blank += string(eol)
			ps.peek()
			continue
		}
		if ps.currentPeek() == eof {
			return blank
		}
		ps.resetPeek(lineStart)
		return blank
	}
}

func (ps *parserStream) skipBlankBlock() string {
	blank := ps.peekBlankBlock()
	ps.skipToPeek()
	return blank
}

func (ps *parserStream) peekBlank() {
	for ps.currentPeek() == ' ' || ps.currentPeek() == eol {
		ps.peek()
	}
}

func (ps *parserStream) skipBlank() {
	ps.peekBlank()
	ps.skipToPeek()
}

func (ps *parserStream) expectChar(ch rune) error {
	if ps.currentChar() == ch {
		ps.next()
		return nil
	}
	return newParseError("E0003", string(ch))
}

func (ps *parserStream) expectLineEnd() error {
	if ps.currentChar() == eof {
		return nil
	}
	if ps.currentChar() == eol {
		ps.next()
		return nil
	}
	// Unicode Character 'SYMBOL FOR NEWLINE' (U+2424)
	return newParseError("E0003", "␤")
}

// takeChar consumes and returns the current char if f(ch) is true. It returns
// (eof, false) at EOF and (0, false) when f rejects the char. The ok return
// distinguishes a consumed char from a rejection.
func (ps *parserStream) takeChar(f func(rune) bool) (rune, bool) {
	ch := ps.currentChar()
	if ch == eof {
		return eof, false
	}
	if f(ch) {
		ps.next()
		return ch, true
	}
	return 0, false
}

func isCharIDStart(ch rune) bool {
	if ch == eof {
		return false
	}
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func (ps *parserStream) isIdentifierStart() bool {
	return isCharIDStart(ps.currentPeek())
}

func (ps *parserStream) isNumberStart() bool {
	var ch rune
	if ps.currentChar() == '-' {
		ch = ps.peek()
	} else {
		ch = ps.currentChar()
	}
	if ch == eof {
		ps.resetPeek(0)
		return false
	}
	isDigit := ch >= '0' && ch <= '9'
	ps.resetPeek(0)
	return isDigit
}

func (ps *parserStream) isCharPatternContinuation(ch rune) bool {
	if ch == eof {
		return false
	}
	return !specialLineStartChars[ch]
}

func (ps *parserStream) isValueStart() bool {
	ch := ps.currentPeek()
	return ch != eol && ch != eof
}

func (ps *parserStream) isValueContinuation() bool {
	column1 := ps.peekOffset
	ps.peekBlankInline()

	if ps.currentPeek() == '{' {
		ps.resetPeek(column1)
		return true
	}

	if ps.peekOffset-column1 == 0 {
		return false
	}

	if ps.isCharPatternContinuation(ps.currentPeek()) {
		ps.resetPeek(column1)
		return true
	}

	return false
}

// isNextLineComment reports whether the next line is a comment of the given
// level. level -1 matches any comment level; 0/1/2 require exactly that depth.
func (ps *parserStream) isNextLineComment(level int) bool {
	if ps.currentChar() != eol {
		return false
	}

	i := 0
	for i <= level || (level == -1 && i < 3) {
		if ps.peek() != '#' {
			if i <= level && level != -1 {
				ps.resetPeek(0)
				return false
			}
			break
		}
		i++
	}

	ch := ps.peek()
	if ch == ' ' || ch == eol {
		ps.resetPeek(0)
		return true
	}

	ps.resetPeek(0)
	return false
}

func (ps *parserStream) isVariantStart() bool {
	currentPeekOffset := ps.peekOffset
	if ps.currentPeek() == '*' {
		ps.peek()
	}
	if ps.currentPeek() == '[' {
		ps.resetPeek(currentPeekOffset)
		return true
	}
	ps.resetPeek(currentPeekOffset)
	return false
}

func (ps *parserStream) isAttributeStart() bool {
	return ps.currentPeek() == '.'
}

func (ps *parserStream) skipToNextEntryStart(junkStart int) {
	lastNewline := ps.lastIndexOfEOL(ps.index)
	if junkStart < lastNewline {
		// Last seen newline is after the junk start: safe to rewind.
		ps.index = lastNewline
	}
	for ps.currentChar() != eof {
		if ps.currentChar() != eol {
			ps.next()
			continue
		}
		first := ps.next()
		if isCharIDStart(first) || first == '-' || first == '#' {
			break
		}
	}
}

func (ps *parserStream) takeIDStart() (rune, error) {
	if isCharIDStart(ps.currentChar()) {
		ret := ps.currentChar()
		ps.next()
		return ret, nil
	}
	return 0, newParseError("E0004", "a-zA-Z")
}

func (ps *parserStream) takeIDChar() (rune, bool) {
	return ps.takeChar(func(ch rune) bool {
		return (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-'
	})
}

func (ps *parserStream) takeDigit() (rune, bool) {
	return ps.takeChar(func(ch rune) bool {
		return ch >= '0' && ch <= '9'
	})
}

func (ps *parserStream) takeHexDigit() (rune, bool) {
	return ps.takeChar(func(ch rune) bool {
		return (ch >= '0' && ch <= '9') ||
			(ch >= 'A' && ch <= 'F') ||
			(ch >= 'a' && ch <= 'f')
	})
}
