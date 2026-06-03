package fluent

import (
	"strings"
	"time"
)

// This file ports fluent.js/fluent-bundle/src/bundle.ts (FluentBundle).

// Function is the signature of a Fluent builtin/runtime function. It receives
// positional and named Value arguments and returns a Value. Returning a non-nil
// error (or panicking) routes through the resolver's fault-tolerant error path,
// rendering `{NAME()}`. Mirrors FluentFunction in fluent.js.
type Function func(positional []Value, named map[string]Value) (Value, error)

// TextTransform transforms the text parts of patterns. Mirrors TextTransform.
type TextTransform func(string) string

// Bundle is a single-language store of translation resources, responsible for
// formatting message values and attributes to strings.
type Bundle struct {
	// locale is the BCP-47 tag used by the pluggable formatters. fluent.js
	// supports a locale fallback list; the core here keeps a single primary
	// locale string (plus the full list for reference) since formatting is
	// delegated to injected formatters.
	locale  string
	locales []string

	terms     map[string]*Term
	messages  map[string]*Message
	functions map[string]Function

	useIsolating bool
	transform    TextTransform

	numberFormatter   NumberFormatter
	dateTimeFormatter DateTimeFormatter
	pluralRules       PluralRules
}

// BundleOption configures a Bundle in NewBundle.
type BundleOption func(*Bundle)

// WithUseIsolating sets whether to wrap interpolations in Unicode isolation
// marks (FSI/PDI). Default is true.
func WithUseIsolating(v bool) BundleOption {
	return func(b *Bundle) { b.useIsolating = v }
}

// WithFunctions registers additional builtin functions, merged over NUMBER and
// DATETIME.
func WithFunctions(fns map[string]Function) BundleOption {
	return func(b *Bundle) {
		for name, fn := range fns {
			b.functions[name] = fn
		}
	}
}

// WithTransform sets the text transform applied to string parts of patterns.
func WithTransform(t TextTransform) BundleOption {
	return func(b *Bundle) {
		if t != nil {
			b.transform = t
		}
	}
}

// WithNumberFormatter injects a NumberFormatter (replaces the no-op default).
func WithNumberFormatter(f NumberFormatter) BundleOption {
	return func(b *Bundle) {
		if f != nil {
			b.numberFormatter = f
		}
	}
}

// WithDateTimeFormatter injects a DateTimeFormatter (replaces the no-op default).
func WithDateTimeFormatter(f DateTimeFormatter) BundleOption {
	return func(b *Bundle) {
		if f != nil {
			b.dateTimeFormatter = f
		}
	}
}

// WithPluralRules injects a PluralRules implementation (replaces the no-op default).
func WithPluralRules(p PluralRules) BundleOption {
	return func(b *Bundle) {
		if p != nil {
			b.pluralRules = p
		}
	}
}

// WithLocales sets the full locale fallback list. The first entry becomes the
// primary locale passed to formatters.
func WithLocales(locales ...string) BundleOption {
	return func(b *Bundle) {
		if len(locales) > 0 {
			b.locales = append([]string(nil), locales...)
			b.locale = locales[0]
		}
	}
}

// NewBundle creates a Bundle for the given primary locale. useIsolating
// defaults to true; NUMBER and DATETIME are always available; the three
// formatters default to the dependency-free no-op implementations.
func NewBundle(locale string, opts ...BundleOption) *Bundle {
	b := &Bundle{
		locale:   locale,
		locales:  []string{locale},
		terms:    make(map[string]*Term),
		messages: make(map[string]*Message),
		functions: map[string]Function{
			"NUMBER":   builtinNUMBER,
			"DATETIME": builtinDATETIME,
		},
		useIsolating:      true,
		transform:         func(s string) string { return s },
		numberFormatter:   defaultNumberFormatter{},
		dateTimeFormatter: defaultDateTimeFormatter{},
		pluralRules:       defaultPluralRules{},
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Locale returns the bundle's primary locale string.
func (b *Bundle) Locale() string { return b.locale }

// AddFunction registers (or overrides) a runtime function by name.
func (b *Bundle) AddFunction(name string, fn Function) {
	b.functions[name] = fn
}

// HasMessage reports whether a public message with the given id exists.
func (b *Bundle) HasMessage(id string) bool {
	_, ok := b.messages[id]
	return ok
}

// GetMessage returns the raw message with the given id, if present.
func (b *Bundle) GetMessage(id string) (*Message, bool) {
	m, ok := b.messages[id]
	return m, ok
}

// AddResource adds a parsed resource to the bundle without allowing overrides.
// It returns errors for any attempted overrides of existing messages/terms.
func (b *Bundle) AddResource(res *Resource) []error {
	return b.addResource(res, false)
}

// AddResourceOverriding adds a parsed resource, allowing it to override existing
// messages and terms.
func (b *Bundle) AddResourceOverriding(res *Resource) []error {
	return b.addResource(res, true)
}

func (b *Bundle) addResource(res *Resource, allowOverrides bool) []error {
	var errors []error

	for _, entry := range res.Body {
		switch e := entry.(type) {
		case *Term:
			if !allowOverrides {
				if _, exists := b.terms[e.ID]; exists {
					errors = append(errors, newOverrideError("term", e.ID))
					continue
				}
			}
			b.terms[e.ID] = e
		case *Message:
			if !allowOverrides {
				if _, exists := b.messages[e.ID]; exists {
					errors = append(errors, newOverrideError("message", e.ID))
					continue
				}
			}
			b.messages[e.ID] = e
		}
	}

	return errors
}

func newOverrideError(kind, id string) error {
	return &overrideError{msg: "Attempt to override an existing " + kind + ": \"" + id + "\""}
}

type overrideError struct{ msg string }

func (e *overrideError) Error() string { return e.msg }

// FormatPattern formats a Pattern to a string. args resolves variable
// references; pass nil for none. errs collects encountered errors; if errs is
// nil, the first error is returned as the fluentPanic-recovered error... but to
// keep a string return, a nil errs causes the first error to be thrown
// (panicked) and recovered into the returned string, matching fluent.js where
// omitting errors throws.
//
// args accepts a map[string]Value (already-typed) — see FormatPatternAny for a
// map[string]any convenience wrapper.
func (b *Bundle) FormatPattern(pattern Pattern, args map[string]Value, errs *[]error) string {
	// Resolve a simple pattern without creating a scope.
	if s, ok := pattern.(string); ok {
		return b.transform(s)
	}

	scope := newScope(b, errs, args)

	var result string
	func() {
		defer func() {
			if r := recover(); r != nil {
				fp, ok := r.(fluentPanic)
				if !ok {
					panic(r)
				}
				if scope.errors != nil {
					*scope.errors = append(*scope.errors, fp.err)
					result = NewNone("").Format(scope)
					return
				}
				// No error sink: rethrow.
				panic(fp)
			}
		}()
		// A nil value (e.g. a message with only attributes) mirrors fluent.js
		// formatPattern(null): resolving a non-array as a complex pattern
		// throws a TypeError, collected and rendered as the {???} fallback.
		if pattern == nil {
			scope.reportError(newTypeError("Cannot format null value"))
			result = NewNone("").Format(scope)
			return
		}
		cp, ok := pattern.(ComplexPattern)
		if !ok {
			result = NewNone("").Format(scope)
			return
		}
		result = resolveComplexPattern(scope, cp).Format(scope)
	}()
	return result
}

// FormatPatternAny is a convenience wrapper accepting raw Go argument values
// (map[string]any). Values are converted to Fluent Values via coerceArg.
func (b *Bundle) FormatPatternAny(pattern Pattern, args map[string]any, errs *[]error) string {
	var typed map[string]Value
	if args != nil {
		typed = make(map[string]Value, len(args))
		for k, v := range args {
			typed[k] = coerceArg(v)
		}
	}
	return b.FormatPattern(pattern, typed, errs)
}

// coerceArg converts a raw Go argument value into a Fluent Value, mirroring the
// JS-value -> FluentValue coercion in resolveVariableReference. Unsupported
// types map to nil (which the resolver reports as a TypeError and renders as a
// missing-variable fallback).
func coerceArg(v any) Value {
	switch x := v.(type) {
	case nil:
		return nil
	case Value:
		return x
	case string:
		return FluentString(x)
	case float64:
		return NewNumber(x, NumberOptions{})
	case float32:
		return NewNumber(float64(x), NumberOptions{})
	case int:
		return NewNumber(float64(x), NumberOptions{})
	case int8:
		return NewNumber(float64(x), NumberOptions{})
	case int16:
		return NewNumber(float64(x), NumberOptions{})
	case int32:
		return NewNumber(float64(x), NumberOptions{})
	case int64:
		return NewNumber(float64(x), NumberOptions{})
	case uint:
		return NewNumber(float64(x), NumberOptions{})
	case uint8:
		return NewNumber(float64(x), NumberOptions{})
	case uint16:
		return NewNumber(float64(x), NumberOptions{})
	case uint32:
		return NewNumber(float64(x), NumberOptions{})
	case uint64:
		return NewNumber(float64(x), NumberOptions{})
	case time.Time:
		return NewDateTime(x, DateTimeOptions{})
	default:
		// Unsupported type (slices, maps, bools, funcs, ...).
		return nil
	}
}

// millisToTime converts a millisecond Unix timestamp to a time.Time (UTC).
func millisToTime(ms float64) time.Time {
	return time.UnixMilli(int64(ms)).UTC()
}

// localesKey joins the locale list for cache/identity purposes (memoizer port).
func localesKey(locales []string) string {
	return strings.Join(locales, " ")
}
