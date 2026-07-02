package fluent

// Scope carries the state of a single pattern resolution: the bundle, the
// caller's arguments, and the errors collected so far. Library users only
// meet it as the argument to Value.Format; its only public surface is Locale.
type Scope struct {
	// bundle is the bundle for which resolution is happening.
	bundle *Bundle
	// errs collects the errors encountered while resolving.
	errs []error
	// args is the dict of developer-provided variables.
	args map[string]Value
	// dirty is the set of complex patterns already entered during this
	// resolution, used to detect cyclic references. Keyed by the address of
	// the pattern's first element, which is stable for a given parsed pattern.
	dirty map[*patternElement]bool
	// params is the dict of parameters passed to a termReference (nil when not
	// inside a term).
	params map[string]Value
	// placeables counts placeables resolved so far, to stop the Billion
	// Laughs and Quadratic Blowup attacks.
	placeables int
}

// Locale returns the locale of the bundle this resolution formats for. It is
// the locale custom Value implementations should render with. A nil scope
// carries no locale context and returns "", so a Value.Format called outside a
// resolution can query it without a nil-pointer panic.
func (s *Scope) Locale() string {
	if s == nil {
		return ""
	}
	return s.bundle.locale
}

// newScope creates a Scope for the given bundle and args.
func newScope(bundle *Bundle, args map[string]Value) *Scope {
	return &Scope{
		bundle: bundle,
		args:   args,
		dirty:  make(map[*patternElement]bool),
	}
}

// reportError records a non-fatal resolution error.
func (s *Scope) reportError(err error) {
	s.errs = append(s.errs, err)
}

// fluentPanic aborts a resolution that must not continue (the placeable
// limit). It is recovered at the FormatPattern boundary.
type fluentPanic struct {
	err error
}
