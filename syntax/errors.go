package syntax

import "fmt"

// ParseError is a recoverable parse error carrying a Fluent error code (E0001..)
// and the formatted human-readable message. During a full Parse these are not
// returned to the caller; instead they become Annotations on Junk entries.
type ParseError struct {
	Code    string
	Args    []any
	Message string
}

func (e *ParseError) Error() string { return e.Message }

// newParseError builds a ParseError, computing its message from the code and
// arguments via getErrorMessage.
func newParseError(code string, args ...any) *ParseError {
	return &ParseError{
		Code:    code,
		Args:    args,
		Message: getErrorMessage(code, args),
	}
}

// getErrorMessage returns the human-readable message for a Fluent error code,
// formatting any arguments. It mirrors getErrorMessage in errors.ts.
func getErrorMessage(code string, args []any) string {
	switch code {
	case "E0001":
		return "Generic error"
	case "E0002":
		return "Expected an entry start"
	case "E0003":
		return fmt.Sprintf("Expected token: \"%v\"", arg(args, 0))
	case "E0004":
		return fmt.Sprintf("Expected a character from range: \"%v\"", arg(args, 0))
	case "E0005":
		return fmt.Sprintf("Expected message \"%v\" to have a value or attributes", arg(args, 0))
	case "E0006":
		return fmt.Sprintf("Expected term \"-%v\" to have a value", arg(args, 0))
	case "E0007":
		return "Keyword cannot end with a whitespace"
	case "E0008":
		return "The callee has to be an upper-case identifier or a term"
	case "E0009":
		return "The argument name has to be a simple identifier"
	case "E0010":
		return "Expected one of the variants to be marked as default (*)"
	case "E0011":
		return "Expected at least one variant after \"->\""
	case "E0012":
		return "Expected value"
	case "E0013":
		return "Expected variant key"
	case "E0014":
		return "Expected literal"
	case "E0015":
		return "Only one variant can be marked as default (*)"
	case "E0016":
		return "Message references cannot be used as selectors"
	case "E0017":
		return "Terms cannot be used as selectors"
	case "E0018":
		return "Attributes of messages cannot be used as selectors"
	case "E0019":
		return "Attributes of terms cannot be used as placeables"
	case "E0020":
		return "Unterminated string expression"
	case "E0021":
		return "Positional arguments must not follow named arguments"
	case "E0022":
		return "Named arguments must be unique"
	case "E0024":
		return "Cannot access variants of a message."
	case "E0025":
		return fmt.Sprintf("Unknown escape sequence: \\%v.", arg(args, 0))
	case "E0026":
		return fmt.Sprintf("Invalid Unicode escape sequence: %v.", arg(args, 0))
	case "E0027":
		return "Unbalanced closing brace in TextElement."
	case "E0028":
		return "Expected an inline expression"
	case "E0029":
		return "Expected simple expression as selector"
	default:
		return code
	}
}

// arg returns the i-th argument or nil if out of range.
func arg(args []any, i int) any {
	if i < len(args) {
		return args[i]
	}
	return nil
}
