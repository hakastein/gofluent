package fluent

import (
	"fmt"
	"strings"
)

// This file ports fluent.js/fluent-bundle/src/resolver.ts.
//
// The resolver formats a Pattern to a Value. It is fault-tolerant: on errors it
// salvages as much of the translation as possible, collecting errors into the
// scope and returning a None when it cannot recover.

// MaxPlaceables is the maximum number of placeables which can be expanded in a
// single FormatPattern call. The limit protects against the Billion Laughs and
// Quadratic Blowup attacks.
const MaxPlaceables = 100

// Unicode bidi isolation characters.
const (
	fsi = "⁨"
	pdi = "⁩"
)

// referenceError / rangeError / typeError mirror the JS error classes so tests
// can distinguish them (errors.As / errors.Is on the typed wrappers).
type referenceError struct{ msg string }

func (e *referenceError) Error() string { return e.msg }

type rangeError struct{ msg string }

func (e *rangeError) Error() string { return e.msg }

type typeError struct{ msg string }

func (e *typeError) Error() string { return e.msg }

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
	// Both are plain strings.
	if ss, ok := selector.(FluentString); ok {
		if ks, ok := key.(FluentString); ok {
			return ss == ks
		}
	}

	selNum, selIsNum := selector.(numberValue)
	keyNum, keyIsNum := key.(numberValue)

	// Both numbers: compare by value (XXX options are not compared, mirroring
	// the fluent.js note).
	if selIsNum && keyIsNum {
		sv, _ := selNum.numberValue()
		kv, _ := keyNum.numberValue()
		if sv == kv {
			return true
		}
	}

	// Numeric selector against a string key: consult the plural rules.
	if selIsNum {
		if ks, ok := key.(FluentString); ok {
			sv, sopts := selNum.numberValue()
			var category string
			if sopts.Type == "ordinal" {
				category = scope.bundle.pluralRules.Ordinal(scope.bundle.locale, sv, sopts)
			} else {
				category = scope.bundle.pluralRules.Cardinal(scope.bundle.locale, sv, sopts)
			}
			if string(ks) == category {
				return true
			}
		}
	}

	return false
}

// getDefault resolves the default variant from a list of variants.
func getDefault(scope *Scope, variants []Variant, star int) Value {
	if star >= 0 && star < len(variants) {
		return resolvePattern(scope, variants[star].Value)
	}
	scope.reportError(newRangeError("No default"))
	return NewNone("")
}

// arguments holds resolved call arguments.
type arguments struct {
	positional []Value
	named      map[string]Value
}

// getArguments resolves arguments to a call expression.
func getArguments(scope *Scope, args []any) arguments {
	positional := []Value{}
	named := make(map[string]Value)

	for _, arg := range args {
		if narg, ok := arg.(*NamedArgument); ok {
			named[narg.Name] = resolveExpression(scope, narg.Value)
		} else {
			positional = append(positional, resolveExpression(scope, arg))
		}
	}

	return arguments{positional: positional, named: named}
}

// resolveExpression resolves an expression to a Value.
func resolveExpression(scope *Scope, expr Expression) Value {
	switch e := expr.(type) {
	case *StringLiteral:
		return FluentString(e.Value)
	case *NumberLiteral:
		return NewNumber(e.Value, NumberOptions{MinimumFractionDigits: intPtr(e.Precision)})
	case *VariableReference:
		return resolveVariableReference(scope, e)
	case *MessageReference:
		return resolveMessageReference(scope, e)
	case *TermReference:
		return resolveTermReference(scope, e)
	case *FunctionReference:
		return resolveFunctionReference(scope, e)
	case *SelectExpression:
		return resolveSelectExpression(scope, e)
	default:
		return NewNone("")
	}
}

// resolveVariableReference resolves a reference to a variable.
func resolveVariableReference(scope *Scope, ref *VariableReference) Value {
	name := ref.Name
	var arg Value
	var found bool

	if scope.paramsSet {
		// Inside a TermReference. It's OK to reference undefined parameters.
		if v, ok := scope.params[name]; ok {
			arg = v
			found = true
		} else {
			return NewNone("$" + name)
		}
	} else if v, ok := scope.args[name]; ok {
		arg = v
		found = true
	}

	if !found {
		scope.reportError(newReferenceError("Unknown variable: $%s", name))
		return NewNone("$" + name)
	}

	// The arg is already a Value (args are normalized on entry). However a nil
	// Value signals an unsupported type that was stored to preserve the key.
	if arg == nil {
		scope.reportError(newTypeError("Variable type not supported: $%s", name))
		return NewNone("$" + name)
	}
	return arg
}

// resolveMessageReference resolves a reference to another message.
func resolveMessageReference(scope *Scope, ref *MessageReference) Value {
	message, ok := scope.bundle.messages[ref.Name]
	if !ok {
		scope.reportError(newReferenceError("Unknown message: %s", ref.Name))
		return NewNone(ref.Name)
	}

	if ref.Attr != "" {
		if attribute, ok := message.Attributes[ref.Attr]; ok {
			return resolvePattern(scope, attribute)
		}
		scope.reportError(newReferenceError("Unknown attribute: %s", ref.Attr))
		return NewNone(ref.Name + "." + ref.Attr)
	}

	if message.Value != nil {
		return resolvePattern(scope, message.Value)
	}

	scope.reportError(newReferenceError("No value: %s", ref.Name))
	return NewNone(ref.Name)
}

// resolveTermReference resolves a call to a Term with key-value arguments.
func resolveTermReference(scope *Scope, ref *TermReference) Value {
	id := "-" + ref.Name
	term, ok := scope.bundle.terms[id]
	if !ok {
		scope.reportError(newReferenceError("Unknown term: %s", id))
		return NewNone(id)
	}

	if ref.Attr != "" {
		if attribute, ok := term.Attributes[ref.Attr]; ok {
			// Every TermReference has its own variables.
			prevParams, prevSet := scope.params, scope.paramsSet
			scope.params = getArguments(scope, ref.Args).named
			scope.paramsSet = true
			resolved := resolvePattern(scope, attribute)
			scope.params, scope.paramsSet = prevParams, prevSet
			return resolved
		}
		scope.reportError(newReferenceError("Unknown attribute: %s", ref.Attr))
		return NewNone(id + "." + ref.Attr)
	}

	prevParams, prevSet := scope.params, scope.paramsSet
	scope.params = getArguments(scope, ref.Args).named
	scope.paramsSet = true
	resolved := resolvePattern(scope, term.Value)
	scope.params, scope.paramsSet = prevParams, prevSet
	return resolved
}

// resolveFunctionReference resolves a call to a Function.
func resolveFunctionReference(scope *Scope, ref *FunctionReference) Value {
	fn, ok := scope.bundle.functions[ref.Name]
	if !ok {
		scope.reportError(newReferenceError("Unknown function: %s()", ref.Name))
		return NewNone(ref.Name + "()")
	}

	resolved := getArguments(scope, ref.Args)
	result, err := callFunction(fn, resolved.positional, resolved.named)
	if err != nil {
		scope.reportError(err)
		return NewNone(ref.Name + "()")
	}
	return result
}

// callFunction invokes a Function, converting a returned error or a Go panic
// into the error path (mirroring the try/catch around func() in fluent.js).
func callFunction(fn Function, positional []Value, named map[string]Value) (result Value, err error) {
	defer func() {
		if r := recover(); r != nil {
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
func resolveSelectExpression(scope *Scope, expr *SelectExpression) Value {
	sel := resolveExpression(scope, expr.Selector)
	if _, isNone := sel.(*None); isNone {
		return getDefault(scope, expr.Variants, expr.Star)
	}

	for _, variant := range expr.Variants {
		key := resolveExpression(scope, variant.Key)
		if matchSelector(scope, sel, key) {
			return resolvePattern(scope, variant.Value)
		}
	}

	return getDefault(scope, expr.Variants, expr.Star)
}

// resolveComplexPattern resolves a complex pattern (text with placeables).
func resolveComplexPattern(scope *Scope, ptn ComplexPattern) Value {
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
		if str, ok := elem.(string); ok {
			sb.WriteString(scope.bundle.transform(str))
			continue
		}

		scope.placeables++
		if scope.placeables > MaxPlaceables {
			if key != nil {
				delete(scope.dirty, key)
			}
			// Fatal error: bail out instantly. The length check protects
			// against excessive memory; "throwing" protects the CPU when long
			// placeables are deeply nested.
			panic(fluentPanic{newRangeError(
				"Too many placeables expanded: %d, max allowed is %d",
				scope.placeables, MaxPlaceables,
			)})
		}

		if useIsolating {
			sb.WriteString(fsi)
		}
		sb.WriteString(resolveExpression(scope, elem).Format(scope))
		if useIsolating {
			sb.WriteString(pdi)
		}
	}

	if key != nil {
		delete(scope.dirty, key)
	}
	return FluentString(sb.String())
}

// patternKey returns a stable identity for a complex pattern, used as the dirty
// set key. It is the address of the pattern's first element. Empty patterns
// have no identity (and cannot be cyclic), so nil is returned.
func patternKey(ptn ComplexPattern) *PatternElement {
	if len(ptn) == 0 {
		return nil
	}
	return &ptn[0]
}

// resolvePattern resolves a simple or complex Pattern to a Value.
func resolvePattern(scope *Scope, value Pattern) Value {
	if value == nil {
		return NewNone("")
	}
	if str, ok := value.(string); ok {
		return FluentString(scope.bundle.transform(str))
	}
	if cp, ok := value.(ComplexPattern); ok {
		return resolveComplexPattern(scope, cp)
	}
	return NewNone("")
}
