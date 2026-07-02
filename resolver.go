package fluent

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// MaxPlaceables is the maximum number of placeables which can be expanded in a
// single FormatPattern call. The limit protects against the Billion Laughs and
// Quadratic Blowup attacks.
const MaxPlaceables = 100

// Unicode bidi isolation characters.
const (
	fsi = "⁨" // FIRST STRONG ISOLATE
	pdi = "⁩" // POP DIRECTIONAL ISOLATE
)

// Error kinds collected by FormatPattern, mirroring the JS error classes
// fluent.js reports (ReferenceError / RangeError / TypeError). Every resolution
// error wraps one of these sentinels, so a caller can classify a failure with
// errors.Is, e.g. errors.Is(err, fluent.ErrReference).
var (
	// ErrReference: an unknown message, term, variable, function, or attribute
	// was referenced.
	ErrReference = errors.New("fluent: reference error")
	// ErrRange: a reference is cyclic, the placeable limit was exceeded, or an
	// option value is invalid.
	ErrRange = errors.New("fluent: range error")
	// ErrType: a value cannot be used in the position it appears (e.g. a
	// non-numeric selector argument, or a term used as a placeable).
	ErrType = errors.New("fluent: type error")
)

// referenceError / rangeError / typeError carry the human-readable message and
// unwrap to the corresponding exported sentinel for errors.Is classification.
type referenceError struct{ msg string }

func (e *referenceError) Error() string { return e.msg }
func (e *referenceError) Unwrap() error { return ErrReference }

type rangeError struct{ msg string }

func (e *rangeError) Error() string { return e.msg }
func (e *rangeError) Unwrap() error { return ErrRange }

type typeError struct{ msg string }

func (e *typeError) Error() string { return e.msg }
func (e *typeError) Unwrap() error { return ErrType }

func newReferenceError(format string, a ...any) *referenceError {
	return &referenceError{msg: fmt.Sprintf(format, a...)}
}
func newRangeError(format string, a ...any) *rangeError {
	return &rangeError{msg: fmt.Sprintf(format, a...)}
}
func newTypeError(format string, a ...any) *typeError {
	return &typeError{msg: fmt.Sprintf(format, a...)}
}

// matchSelector matches a variant key against the given selector.
func matchSelector(scope *Scope, selector, key Value) bool {
	if ss, ok := selector.(String); ok {
		ks, ok := key.(String)
		return ok && ss == ks
	}

	sel, ok := selector.(*Number)
	if !ok {
		return false
	}

	if kn, ok := key.(*Number); ok {
		return sel.Value == kn.Value
	}

	// Numeric selector against a string key: consult the plural rules.
	ks, ok := key.(String)
	if !ok {
		return false
	}
	var category string
	if sel.Options.Type == "ordinal" {
		category = scope.bundle.pluralRules.Ordinal(scope.bundle.locale, sel.Value, sel.Options)
	} else {
		category = scope.bundle.pluralRules.Cardinal(scope.bundle.locale, sel.Value, sel.Options)
	}
	return string(ks) == category
}

// getDefault resolves the default variant. The runtime parser is the only
// producer of selectExpressions and rejects sources without a valid default,
// so star always indexes variants.
func getDefault(scope *Scope, variants []variant, star int) Value {
	return resolvePattern(scope, variants[star].value)
}

// resolveArguments resolves the arguments of a term or function call. named is
// never nil: Scope.params relies on nil meaning "not inside a term".
func resolveArguments(scope *Scope, args callArguments) (positional []Value, named map[string]Value) {
	for _, arg := range args.positional {
		positional = append(positional, resolveExpression(scope, arg))
	}
	named = make(map[string]Value, len(args.named))
	for _, arg := range args.named {
		named[arg.name] = resolveExpression(scope, arg.value)
	}
	return positional, named
}

// resolveExpression resolves an expression to a Value.
func resolveExpression(scope *Scope, expr expression) Value {
	switch e := expr.(type) {
	case *stringLiteral:
		return String(e.value)
	case *numberLiteral:
		return NewNumber(e.value, NumberOptions{MinimumFractionDigits: intPtr(e.precision)})
	case *variableReference:
		return resolveVariableReference(scope, e)
	case *messageReference:
		return resolveMessageReference(scope, e)
	case *termReference:
		return resolveTermReference(scope, e)
	case *functionReference:
		return resolveFunctionReference(scope, e)
	case *selectExpression:
		return resolveSelectExpression(scope, e)
	default:
		return NewNone("")
	}
}

// resolveVariableReference resolves a reference to a variable.
func resolveVariableReference(scope *Scope, ref *variableReference) Value {
	name := ref.name

	// Inside a termReference the params replace the args entirely, and it's OK
	// to reference undefined parameters: no error is reported.
	if scope.params != nil {
		if v, ok := scope.params[name]; ok {
			return v
		}
		return NewNone("$" + name)
	}

	arg, ok := scope.args[name]
	if !ok {
		scope.reportError(newReferenceError("Unknown variable: $%s", name))
		return NewNone("$" + name)
	}
	// Args are normalized to Values on entry; a nil Value marks an argument
	// of an unsupported Go type whose key was preserved.
	if arg == nil {
		scope.reportError(newTypeError("Variable type not supported: $%s", name))
		return NewNone("$" + name)
	}
	return arg
}

// resolveMessageReference resolves a reference to another message.
func resolveMessageReference(scope *Scope, ref *messageReference) Value {
	message, ok := scope.bundle.Message(ref.name)
	if !ok {
		scope.reportError(newReferenceError("Unknown message: %s", ref.name))
		return NewNone(ref.name)
	}

	if ref.attr != "" {
		if attribute, ok := message.Attributes[ref.attr]; ok {
			return resolvePattern(scope, attribute)
		}
		scope.reportError(newReferenceError("Unknown attribute: %s", ref.attr))
		return NewNone(ref.name + "." + ref.attr)
	}

	if message.Value != nil {
		return resolvePattern(scope, message.Value)
	}

	scope.reportError(newReferenceError("No value: %s", ref.name))
	return NewNone(ref.name)
}

// withTermParams resolves pattern with the term's own parameters installed.
// The params are cleared afterwards rather than restored (matching fluent.js):
// a variable referenced after an embedded term resolves against the top-level
// args.
func withTermParams(scope *Scope, args callArguments, pattern Pattern) Value {
	_, scope.params = resolveArguments(scope, args)
	resolved := resolvePattern(scope, pattern)
	scope.params = nil
	return resolved
}

// resolveTermReference resolves a call to a term with key-value arguments.
func resolveTermReference(scope *Scope, ref *termReference) Value {
	id := "-" + ref.name
	t, ok := scope.bundle.lookupTerm(id)
	if !ok {
		scope.reportError(newReferenceError("Unknown term: %s", id))
		return NewNone(id)
	}

	if ref.attr != "" {
		if attribute, ok := t.attributes[ref.attr]; ok {
			return withTermParams(scope, ref.args, attribute)
		}
		scope.reportError(newReferenceError("Unknown attribute: %s", ref.attr))
		return NewNone(id + "." + ref.attr)
	}

	return withTermParams(scope, ref.args, t.value)
}

// resolveFunctionReference resolves a call to a Function.
func resolveFunctionReference(scope *Scope, ref *functionReference) Value {
	fn, ok := scope.bundle.lookupFunction(ref.name)
	if !ok {
		scope.reportError(newReferenceError("Unknown function: %s()", ref.name))
		return NewNone(ref.name + "()")
	}

	positional, named := resolveArguments(scope, ref.args)
	result, err := callFunction(fn, positional, named)
	if err != nil {
		scope.reportError(err)
		return NewNone(ref.name + "()")
	}
	return result
}

// callFunction invokes a Function, converting a returned error or a panic into
// the error path. A runtime.Error is a programming bug, not a translation
// error, so it is re-panicked instead.
func callFunction(fn Function, positional []Value, named map[string]Value) (result Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(runtime.Error); ok {
				panic(re)
			}
			if fe, ok := r.(error); ok {
				err = fe
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	return fn(positional, named)
}

// resolveSelectExpression resolves a select expression to the member value.
func resolveSelectExpression(scope *Scope, expr *selectExpression) Value {
	sel := resolveExpression(scope, expr.selector)
	if _, isNone := sel.(*None); isNone {
		return getDefault(scope, expr.variants, expr.star)
	}

	for _, v := range expr.variants {
		key := resolveExpression(scope, v.key)
		if matchSelector(scope, sel, key) {
			return resolvePattern(scope, v.value)
		}
	}

	return getDefault(scope, expr.variants, expr.star)
}

// resolveComplexPattern resolves a complex pattern (text with placeables).
func resolveComplexPattern(scope *Scope, ptn complexPattern) Value {
	key := patternKey(ptn)
	if key != nil && scope.dirty[key] {
		scope.reportError(newRangeError("Cyclic reference"))
		return NewNone("")
	}

	if key != nil {
		scope.dirty[key] = true
	}

	var sb strings.Builder

	// Wrap interpolations with directional isolation marks only when the
	// pattern has more than one element.
	useIsolating := scope.bundle.useIsolating && len(ptn) > 1

	for _, elem := range ptn {
		if text, ok := elem.(textElement); ok {
			sb.WriteString(scope.bundle.transform(string(text)))
			continue
		}

		scope.placeables++
		if scope.placeables > MaxPlaceables {
			// Fatal: abort the whole resolution, recovered at the
			// FormatPattern boundary.
			panic(fluentPanic{newRangeError(
				"Too many placeables expanded: %d, max allowed is %d",
				scope.placeables, MaxPlaceables,
			)})
		}

		expr, ok := elem.(expression)
		if !ok {
			continue
		}
		if useIsolating {
			sb.WriteString(fsi)
		}
		sb.WriteString(resolveExpression(scope, expr).Format(scope))
		if useIsolating {
			sb.WriteString(pdi)
		}
	}

	if key != nil {
		delete(scope.dirty, key)
	}
	return String(sb.String())
}

// patternKey returns a stable identity for a complex pattern, used as the dirty
// set key. It is the address of the pattern's first element. Empty patterns
// have no identity (and cannot be cyclic), so nil is returned.
func patternKey(ptn complexPattern) *patternElement {
	if len(ptn) == 0 {
		return nil
	}
	return &ptn[0]
}

// resolvePattern resolves a simple or complex Pattern to a Value.
func resolvePattern(scope *Scope, value Pattern) Value {
	switch p := value.(type) {
	case nil:
		return NewNone("")
	case textPattern:
		return String(scope.bundle.transform(string(p)))
	case complexPattern:
		return resolveComplexPattern(scope, p)
	default:
		return NewNone("")
	}
}
