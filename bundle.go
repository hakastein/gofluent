package fluent

import (
	"sync"
	"time"
)

// Function is the signature of a Fluent builtin/runtime function. It receives
// positional and named Value arguments and returns a Value. Returning a non-nil
// error (or panicking) routes through the resolver's fault-tolerant error path,
// rendering `{NAME()}`. Mirrors FluentFunction in fluent.js.
type Function func(positional []Value, named map[string]Value) (Value, error)

// TextTransform transforms the text parts of patterns.
type TextTransform func(string) string

// Bundle is a single-language store of translation resources, responsible for
// formatting message values and attributes to strings.
//
// A Bundle is safe for concurrent use: FormatPattern, Message, AddFunction,
// AddResource, and AddResourceOverriding may be called from multiple
// goroutines simultaneously. The locale and the injected formatters are set
// once at construction and never mutated afterwards.
type Bundle struct {
	// locale is the primary BCP-47 tag passed to the pluggable formatters;
	// locales keeps the full fallback list set by WithLocales.
	locale  string
	locales []string

	// mu guards terms, messages, and functions. It is held per map operation,
	// never across a whole FormatPattern or a user-function call, so a function
	// that itself calls AddFunction does not deadlock.
	mu        sync.RWMutex
	terms     map[string]*term
	messages  map[string]*Message
	functions map[string]Function

	useIsolating bool
	transform    TextTransform

	numberFormatter   NumberFormatter
	dateTimeFormatter DateTimeFormatter
	pluralRules       PluralRules
}

// Option configures a Bundle in NewBundle.
type Option func(*Bundle)

// WithUseIsolating sets whether to wrap interpolations in Unicode isolation
// marks (FSI/PDI). Default is true.
func WithUseIsolating(v bool) Option {
	return func(b *Bundle) { b.useIsolating = v }
}

// WithFunctions registers additional builtin functions, merged over NUMBER and
// DATETIME.
func WithFunctions(fns map[string]Function) Option {
	return func(b *Bundle) {
		for name, fn := range fns {
			b.functions[name] = fn
		}
	}
}

// WithTransform sets the text transform applied to string parts of patterns.
func WithTransform(t TextTransform) Option {
	return func(b *Bundle) {
		if t != nil {
			b.transform = t
		}
	}
}

// WithNumberFormatter injects a NumberFormatter (replaces the CLDR-backed default).
func WithNumberFormatter(f NumberFormatter) Option {
	return func(b *Bundle) {
		if f != nil {
			b.numberFormatter = f
		}
	}
}

// WithDateTimeFormatter injects a DateTimeFormatter (replaces the CLDR-backed default).
func WithDateTimeFormatter(f DateTimeFormatter) Option {
	return func(b *Bundle) {
		if f != nil {
			b.dateTimeFormatter = f
		}
	}
}

// WithPluralRules injects a PluralRules implementation (replaces the CLDR-backed default).
func WithPluralRules(p PluralRules) Option {
	return func(b *Bundle) {
		if p != nil {
			b.pluralRules = p
		}
	}
}

// WithLocales sets the full locale fallback list. The first entry becomes the
// primary locale passed to formatters.
func WithLocales(locales ...string) Option {
	return func(b *Bundle) {
		if len(locales) > 0 {
			b.locales = append([]string(nil), locales...)
			b.locale = locales[0]
		}
	}
}

// NewBundle creates a Bundle for the given primary locale. useIsolating
// defaults to true; NUMBER and DATETIME are always available; the three
// formatters default to the CLDR-backed implementations (matching Intl.*).
// Applications must blank-import the locale data they format, e.g.
// import _ "github.com/hakastein/gocldr/locales/ru" (or .../locales/all);
// with none imported, formatting degrades to the CLDR root / RFC 3339.
func NewBundle(locale string, opts ...Option) *Bundle {
	b := &Bundle{
		locale:   locale,
		locales:  []string{locale},
		terms:    make(map[string]*term),
		messages: make(map[string]*Message),
		functions: map[string]Function{
			"NUMBER":   builtinNUMBER,
			"DATETIME": builtinDATETIME,
		},
		useIsolating:      true,
		transform:         func(s string) string { return s },
		numberFormatter:   cldrNumberFormatter{},
		dateTimeFormatter: cldrDateTimeFormatter{},
		pluralRules:       cldrPluralRules{},
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
	b.mu.Lock()
	b.functions[name] = fn
	b.mu.Unlock()
}

// Message returns the message with the given id, if present.
func (b *Bundle) Message(id string) (*Message, bool) {
	return b.lookupMessage(id)
}

// lookupMessage returns the message with the given id under a read lock.
func (b *Bundle) lookupMessage(id string) (*Message, bool) {
	b.mu.RLock()
	m, ok := b.messages[id]
	b.mu.RUnlock()
	return m, ok
}

// lookupTerm returns the term with the given id (including the leading "-")
// under a read lock.
func (b *Bundle) lookupTerm(id string) (*term, bool) {
	b.mu.RLock()
	t, ok := b.terms[id]
	b.mu.RUnlock()
	return t, ok
}

// lookupFunction returns the function registered under name under a read lock.
func (b *Bundle) lookupFunction(name string) (Function, bool) {
	b.mu.RLock()
	fn, ok := b.functions[name]
	b.mu.RUnlock()
	return fn, ok
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
	var errs []error

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, e := range res.entries {
		switch e := e.(type) {
		case *term:
			if !allowOverrides {
				if _, exists := b.terms[e.id]; exists {
					errs = append(errs, newOverrideError("term", e.id))
					continue
				}
			}
			b.terms[e.id] = e
		case *Message:
			if !allowOverrides {
				if _, exists := b.messages[e.ID]; exists {
					errs = append(errs, newOverrideError("message", e.ID))
					continue
				}
			}
			b.messages[e.ID] = e
		}
	}

	return errs
}

func newOverrideError(kind, id string) error {
	return &overrideError{msg: "Attempt to override an existing " + kind + ": \"" + id + "\""}
}

type overrideError struct{ msg string }

func (e *overrideError) Error() string { return e.msg }

// FormatPattern formats a Pattern — a Message value or attribute — to a
// string. args resolves variable references; pass nil for none. Argument
// values may be strings, any integer or float type, time.Time, or a Value
// (e.g. a Number carrying formatting options); other types render as a
// missing-variable fallback with an error.
//
// Formatting is fault-tolerant: a best-effort string is always returned, and
// every problem encountered (missing references, type mismatches, ...) is
// reported in errs, each classified by one of the ErrReference / ErrRange /
// ErrType sentinels.
//
// Precision note: integer arguments are stored as float64 (Fluent's only
// numeric type, matching JS). int64/uint64 magnitudes above 2^53 cannot be
// represented exactly and may be rounded; pass a preformatted string (or a
// custom Value) when exact rendering of such large integers matters.
func (b *Bundle) FormatPattern(pattern Pattern, args map[string]any) (result string, errs []error) {
	// A simple pattern resolves without a scope.
	if s, ok := pattern.(textPattern); ok {
		return b.transform(string(s)), nil
	}

	scope := newScope(b, coerceArgs(args))

	defer func() {
		if r := recover(); r != nil {
			fp, ok := r.(fluentPanic)
			if !ok {
				panic(r)
			}
			scope.reportError(fp.err)
			result = NewNone("").Format(scope)
			errs = scope.errs
		}
	}()

	// A nil pattern (e.g. a message with only attributes) mirrors fluent.js
	// formatPattern(null): a type error, rendered as the {???} fallback.
	if pattern == nil {
		scope.reportError(newTypeError("Cannot format null value"))
		return NewNone("").Format(scope), scope.errs
	}
	cp, ok := pattern.(complexPattern)
	if !ok {
		return NewNone("").Format(scope), scope.errs
	}
	return resolveComplexPattern(scope, cp).Format(scope), scope.errs
}

// coerceArgs converts raw Go argument values into Fluent Values.
func coerceArgs(args map[string]any) map[string]Value {
	if args == nil {
		return nil
	}
	typed := make(map[string]Value, len(args))
	for k, v := range args {
		typed[k] = coerceArg(v)
	}
	return typed
}

// coerceArg converts a raw Go argument value into a Fluent Value. Unsupported
// types map to nil, which the resolver reports as a type error and renders as
// a missing-variable fallback.
func coerceArg(v any) Value {
	switch x := v.(type) {
	case nil:
		return nil
	case Value:
		return x
	case string:
		return String(x)
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
