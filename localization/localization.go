// Package localization is the high-level localization layer for gofluent. It
// formats messages across an ordered chain of locale Bundles with fallback:
// the first bundle that resolves a message wins. Bundles are built eagerly
// and formatting is synchronous. It mirrors fluent.js Localization combined
// with @fluent/sequence's mapBundleSync.
//
// This package depends only on the core fluent package and the standard
// library. Bundles inherit the core's default CLDR formatters; override them
// (or set other options) through Config.BundleOptions.
package localization

import (
	"fmt"
	"slices"
	"strings"

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/langneg"
)

// Localization holds an ordered slice of bundles, highest-priority locale
// first. Formatting walks the chain and returns the first resolution.
type Localization struct {
	bundles []*fluent.Bundle
}

// ResourceLoader loads the FTL source for a (locale, resourceID) pair. It
// returns an error if the resource cannot be found or read.
type ResourceLoader func(locale, resourceID string) (string, error)

// New builds a Localization from already-constructed bundles, in priority order
// (highest-priority locale first). Callers retain ownership of the input slice.
func New(bundles []*fluent.Bundle) *Localization {
	return &Localization{bundles: slices.Clone(bundles)}
}

// Bundles returns the localization's bundles in priority order. The returned
// slice is a copy.
func (l *Localization) Bundles() []*fluent.Bundle {
	return slices.Clone(l.bundles)
}

// Config configures NewFromLocales.
type Config struct {
	// Requested is the user's locale preference list, in priority order.
	Requested []string
	// Available is the set of locales the application ships.
	Available []string
	// Default is appended as the last-resort locale. langneg.Lookup requires
	// it to be non-empty.
	Default string
	// Strategy selects the negotiation algorithm. The zero value is
	// langneg.Filtering.
	Strategy langneg.Strategy
	// Resources lists the resource ids loaded into every bundle.
	Resources []string
	// Loader resolves a (locale, resource id) pair to FTL source.
	Loader ResourceLoader
	// BundleOptions are forwarded to every fluent.NewBundle call, e.g. to
	// override the default formatters per bundle.
	BundleOptions []fluent.Option
}

// NewFromLocales negotiates the supported locales out of cfg.Requested and
// cfg.Available against cfg.Default, then builds one Bundle per negotiated
// locale by loading every cfg.Resources id through cfg.Loader. Bundles are
// built in negotiated (priority) order.
//
// Loader and AddResource errors are collected and returned but are non-fatal:
// a failing resource is skipped so the rest of the chain still works, and a
// bundle is created for each negotiated locale regardless. A negotiation
// error is fatal: the Localization is nil.
func NewFromLocales(cfg Config) (*Localization, []error) {
	supported, err := langneg.NegotiateLanguages(cfg.Requested, cfg.Available, cfg.Default, cfg.Strategy)
	if err != nil {
		return nil, []error{err}
	}

	var errs []error
	bundles := make([]*fluent.Bundle, 0, len(supported))
	for _, locale := range supported {
		bundle := fluent.NewBundle(locale, cfg.BundleOptions...)
		for _, resID := range cfg.Resources {
			source, loadErr := cfg.Loader(locale, resID)
			if loadErr != nil {
				errs = append(errs, fmt.Errorf("localization: loading %q for %q: %w", resID, locale, loadErr))
				continue
			}
			for _, ae := range bundle.AddResource(fluent.NewResource(source)) {
				errs = append(errs, fmt.Errorf("localization: adding %q to %q: %w", resID, locale, ae))
			}
		}
		bundles = append(bundles, bundle)
	}

	return New(bundles), errs
}

// splitID splits a dotted id "msg.attr" into ("msg", "attr"). A bare "msg"
// yields ("msg", ""). Only the first dot is significant.
func splitID(id string) (msgID, attr string) {
	if i := strings.IndexByte(id, '.'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return id, ""
}

// FormatValue formats the message (or attribute) identified by id, walking the
// bundle chain in priority order. id may be "msg" or "msg.attr". A bundle is
// skipped when its message has no value (value form) or lacks the requested
// attribute (attribute form). Resolver errors encountered in the winning
// bundle are returned. On a total miss the id is returned unchanged together
// with a single *NotFoundError.
//
// This mirrors fluent.js Localization.formatValue layered over the fluent-dom
// dotted attribute syntax.
func (l *Localization) FormatValue(id string, args map[string]any) (string, []error) {
	msgID, attr := splitID(id)

	for _, bundle := range l.bundles {
		msg, ok := bundle.Message(msgID)
		if !ok {
			continue
		}

		if attr == "" {
			// A message with attributes but no value cannot produce a string
			// value; treat it as a miss so the next bundle gets a chance.
			if msg.Value == nil {
				continue
			}
			return bundle.FormatPattern(msg.Value, args)
		}

		pattern, has := msg.Attributes[attr]
		if !has {
			// Message exists here but lacks the requested attribute; fall through
			// to the next bundle (fluent.js missing-attribute fallback).
			continue
		}
		return bundle.FormatPattern(pattern, args)
	}

	return id, []error{&NotFoundError{ID: id}}
}

// Message is a fully formatted message: its value and all of its attributes.
type Message struct {
	Value      string
	Attributes map[string]string
}

// FormatMessage resolves a whole message (value plus all attributes) from the
// first bundle in the chain that defines it. The value is "" when the message
// has only attributes. On a total miss the value is the id and the errors
// contain a single *NotFoundError.
func (l *Localization) FormatMessage(id string, args map[string]any) (Message, []error) {
	for _, bundle := range l.bundles {
		msg, ok := bundle.Message(id)
		if !ok {
			continue
		}

		var out Message
		var errs []error
		if msg.Value != nil {
			var ferrs []error
			out.Value, ferrs = bundle.FormatPattern(msg.Value, args)
			errs = append(errs, ferrs...)
		}
		if len(msg.Attributes) > 0 {
			out.Attributes = make(map[string]string, len(msg.Attributes))
			for name, pattern := range msg.Attributes {
				formatted, ferrs := bundle.FormatPattern(pattern, args)
				out.Attributes[name] = formatted
				errs = append(errs, ferrs...)
			}
		}
		return out, errs
	}

	return Message{Value: id}, []error{&NotFoundError{ID: id}}
}

// NotFoundError reports that an id could not be resolved in any bundle.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("localization: no bundle resolved id %q", e.ID)
}
