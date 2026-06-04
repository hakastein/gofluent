// Package localization is the high-level, synchronous localization layer for
// gofluent. It formats messages across an ordered chain of locale Bundles with
// fallback, mirroring the behavior of fluent.js Localization combined with
// @fluent/sequence's mapBundleSync (first bundle that has the message wins).
//
// Unlike fluent.js, this layer is synchronous: Go's use case and the ergonomics
// of the language make the async-iterator machinery unnecessary. Bundles are
// built eagerly.
//
// This package depends only on the core fluent package and the standard
// library; it does not import fluentx. Callers wire pluggable formatters by
// passing fluent.BundleOption values through to the bundle builders.
package localization

import (
	"fmt"
	"strings"

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/langneg"
)

// Localization holds an ordered slice of bundles, highest-priority locale
// first. Formatting walks the chain and returns the first resolution.
type Localization struct {
	bundles []*fluent.Bundle
}

// L10nID is a single batch request: a message id (optionally dotted with an
// attribute, "msg.attr") plus its formatting arguments.
type L10nID struct {
	ID   string
	Args map[string]any
}

// ResourceLoader loads the FTL source for a (locale, resourceID) pair. It
// returns an error if the resource cannot be found or read; missing resources
// are tolerated (the resulting bundle simply lacks those messages).
type ResourceLoader func(locale, resourceID string) (string, error)

// New builds a Localization from already-constructed bundles, in priority order
// (highest-priority locale first). The slice is copied.
func New(bundles []*fluent.Bundle) *Localization {
	cp := make([]*fluent.Bundle, len(bundles))
	copy(cp, bundles)
	return &Localization{bundles: cp}
}

// Bundles returns the localization's bundles in priority order. The returned
// slice is a copy.
func (l *Localization) Bundles() []*fluent.Bundle {
	cp := make([]*fluent.Bundle, len(l.bundles))
	copy(cp, l.bundles)
	return cp
}

// NewFromLocales is the higher-level constructor. It negotiates the supported
// locales out of requested/available against defaultLocale (Filtering
// strategy), then builds one Bundle per negotiated locale by loading every
// resourceID through loader and AddResource-ing it. Bundles are returned in
// negotiated (priority) order.
//
// bundleOpts are forwarded to every fluent.NewBundle call, so callers can wire
// fluentx (or any other) formatters per bundle.
//
// Loader and parse errors are collected and returned but are non-fatal: a
// failing resource is skipped so the rest of the chain still works. A bundle is
// always created for each negotiated locale even if some of its resources fail
// to load.
func NewFromLocales(
	requested, available []string,
	defaultLocale string,
	resourceIDs []string,
	loader ResourceLoader,
	bundleOpts ...fluent.BundleOption,
) (*Localization, []error) {
	return NewFromLocalesStrategy(requested, available, defaultLocale, langneg.Filtering, resourceIDs, loader, bundleOpts...)
}

// NewFromLocalesStrategy is NewFromLocales with an explicit negotiation
// strategy.
func NewFromLocalesStrategy(
	requested, available []string,
	defaultLocale string,
	strategy langneg.Strategy,
	resourceIDs []string,
	loader ResourceLoader,
	bundleOpts ...fluent.BundleOption,
) (*Localization, []error) {
	var errs []error

	supported, err := langneg.NegotiateLanguagesErr(requested, available, defaultLocale, strategy)
	if err != nil {
		return New(nil), []error{err}
	}

	bundles := make([]*fluent.Bundle, 0, len(supported))
	for _, locale := range supported {
		bundle := fluent.NewBundle(locale, bundleOpts...)
		for _, resID := range resourceIDs {
			source, loadErr := loader(locale, resID)
			if loadErr != nil {
				errs = append(errs, fmt.Errorf("localization: loading %q for %q: %w", resID, locale, loadErr))
				continue
			}
			res, parseErrs := fluent.NewResource(source)
			for _, pe := range parseErrs {
				errs = append(errs, fmt.Errorf("localization: parsing %q for %q: %w", resID, locale, pe))
			}
			if res == nil {
				continue
			}
			for _, ae := range bundle.AddResource(res) {
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
// bundle chain in priority order. id may be "msg" or "msg.attr". The first
// bundle whose message — and, for the dotted form, whose attribute — resolves
// wins. Resolver errors encountered in the winning bundle are returned. On a
// total miss the id is returned unchanged together with a single
// not-found error.
//
// This mirrors fluent.js Localization.formatValue (via formatValues) layered
// over the fluent-dom dotted attribute syntax.
func (l *Localization) FormatValue(id string, args map[string]any) (string, []error) {
	msgID, attr := splitID(id)

	for _, bundle := range l.bundles {
		msg, ok := bundle.GetMessage(msgID)
		if !ok {
			continue
		}

		if attr == "" {
			// A message with attributes but no value cannot produce a string
			// value; treat it as a miss so the next bundle gets a chance.
			if msg.Value == nil {
				continue
			}
			var errs []error
			out := bundle.FormatPatternAny(msg.Value, args, &errs)
			return out, errs
		}

		pattern, has := msg.Attributes[attr]
		if !has {
			// Message exists here but lacks the requested attribute; fall through
			// to the next bundle (fluent.js missing-attribute fallback).
			continue
		}
		var errs []error
		out := bundle.FormatPatternAny(pattern, args, &errs)
		return out, errs
	}

	return id, []error{&NotFoundError{ID: id}}
}

// FormatValues formats a batch of ids, each with its own args. The returned
// slice is positional (one entry per input). Errors from all entries are
// concatenated. Mirrors fluent.js Localization.formatValues.
func (l *Localization) FormatValues(ids []L10nID) ([]string, []error) {
	out := make([]string, len(ids))
	var errs []error
	for i, key := range ids {
		val, e := l.FormatValue(key.ID, key.Args)
		out[i] = val
		errs = append(errs, e...)
	}
	return out, errs
}

// FormatMessage resolves a whole message (value plus all attributes) from the
// first bundle in the chain that defines it. found reports whether any bundle
// had the message; when false value is the id and attributes is nil. value is
// "" when the message has only attributes and no value. Mirrors fluent.js
// messageFromBundle layered over formatWithFallback.
func (l *Localization) FormatMessage(id string, args map[string]any) (value string, attributes map[string]string, found bool, errs []error) {
	for _, bundle := range l.bundles {
		msg, ok := bundle.GetMessage(id)
		if !ok {
			continue
		}

		if msg.Value != nil {
			value = bundle.FormatPatternAny(msg.Value, args, &errs)
		}
		if len(msg.Attributes) > 0 {
			attributes = make(map[string]string, len(msg.Attributes))
			for name, pattern := range msg.Attributes {
				attributes[name] = bundle.FormatPatternAny(pattern, args, &errs)
			}
		}
		return value, attributes, true, errs
	}

	return id, nil, false, []error{&NotFoundError{ID: id}}
}

// NotFoundError reports that an id could not be resolved in any bundle.
type NotFoundError struct {
	ID string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("localization: no bundle resolved id %q", e.ID)
}
