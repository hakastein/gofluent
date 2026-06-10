package fluent

// Scope stores the data required for a single pattern resolution and for error
// recovery. A new Scope is created per FormatPattern call on a complex pattern.
type Scope struct {
	// bundle is the bundle for which resolution is happening.
	bundle *Bundle
	// errors is the list of errors collected while resolving. If nil, the
	// resolver throws (panics) on the first error instead of collecting.
	errors *[]error
	// args is the dict of developer-provided variables.
	args map[string]Value
	// dirty is the set of complex patterns already encountered during this
	// resolution. Used to detect and prevent cyclic resolutions. Keyed by the
	// pointer identity of the underlying ComplexPattern's backing array (the
	// address of its first element), which is stable for a given parsed pattern.
	dirty map[*PatternElement]bool
	// params is the dict of parameters passed to a TermReference (or nil when
	// not inside a term).
	params    map[string]Value
	paramsSet bool
	// placeables is the running count of placeables resolved so far. Used to
	// detect the Billion Laughs and Quadratic Blowup attacks.
	placeables int
}

// newScope creates a Scope for the given bundle, error sink, and args.
func newScope(bundle *Bundle, errors *[]error, args map[string]Value) *Scope {
	return &Scope{
		bundle: bundle,
		errors: errors,
		args:   args,
		dirty:  make(map[*PatternElement]bool),
	}
}

// reportError records an error. If the scope has no error sink, the error is
// "thrown" by panicking with a fluentPanic, mirroring fluent.js where an
// absent errors array causes reportError to rethrow.
func (s *Scope) reportError(err error) {
	if s.errors == nil {
		panic(fluentPanic{err})
	}
	*s.errors = append(*s.errors, err)
}

// fluentPanic wraps an error that is "thrown" out of resolution when no error
// sink is provided. It is recovered at the FormatPattern boundary.
type fluentPanic struct {
	err error
}
