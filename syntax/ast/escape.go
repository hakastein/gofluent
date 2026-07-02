package ast

import "strconv"

// unescapeStringLiteral resolves the known Fluent escape sequences in a string
// literal's raw value: \\, \", \uHHHH and \UHHHHHH. Escape sequences for
// surrogate code points are replaced with U+FFFD, matching the reference.
func unescapeStringLiteral(s string) string {
	out := make([]rune, 0, len(s))
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] != '\\' || i+1 >= len(rs) {
			out = append(out, rs[i])
			continue
		}
		switch rs[i+1] {
		case '\\':
			out = append(out, '\\')
			i++
		case '"':
			out = append(out, '"')
			i++
		case 'u', 'U':
			n := 4
			if rs[i+1] == 'U' {
				n = 6
			}
			if r, ok := parseHexEscape(rs, i+2, n); ok {
				out = append(out, r)
				i += 1 + n
			} else {
				out = append(out, rs[i])
			}
		default:
			out = append(out, rs[i])
		}
	}
	return string(out)
}

// parseHexEscape reads n hex digits starting at index start, returning the
// resolved rune. Surrogate code points and code points beyond the Unicode
// maximum (U+10FFFF) are intentionally replaced with U+FFFD, matching the
// reference (where String.fromCodePoint of an out-of-range value would throw,
// so such escapes never produce a valid scalar).
func parseHexEscape(rs []rune, start, n int) (rune, bool) {
	if start+n > len(rs) {
		return 0, false
	}
	for _, r := range rs[start : start+n] {
		// strconv.ParseInt would also accept a leading sign; only hex digits
		// form a valid escape.
		switch {
		case '0' <= r && r <= '9', 'a' <= r && r <= 'f', 'A' <= r && r <= 'F':
		default:
			return 0, false
		}
	}
	cp, err := strconv.ParseInt(string(rs[start:start+n]), 16, 32)
	if err != nil {
		return 0, false
	}
	if (cp >= 0xD800 && cp <= 0xDFFF) || cp > 0x10FFFF {
		return '�', true
	}
	return rune(cp), true
}
