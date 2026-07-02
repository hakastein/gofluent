package conformance

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This suite cross-checks the two hand-written FTL parsers against each other
// over the upstream reference fixtures:
//
//   - the runtime parser (fluent.NewResource), which feeds the resolver in
//     production, and
//   - the syntax parser (syntax.Parse), which the reference conformance suite
//     already validates against the vendored fixtures.
//
// The syntax parser is the trusted oracle. Each fixture is parsed by both; the
// runtime parser's exposed message/attribute inventory and its formatted text
// must agree with the syntax AST. A divergence in the runtime parser's
// indent/dedent or entry-boundary handling would otherwise ship wrong output
// while the existing (syntax-only) conformance suite stays green.
//
// Scope and known asymmetry: the runtime parser is deliberately fault-tolerant
// and lossy (it matches fluent.js's own runtime parser). It silently skips
// broken entries instead of emitting Junk, drops comments, and is more lenient
// than the syntax parser about a few argument-list edge cases — so it can
// accept an entry the syntax parser turns into Junk. Because Bundle exposes no
// enumeration of its messages, the checks here are driven from the syntax AST
// (the forward direction: "everything syntax accepts, runtime must expose the
// same way"). That is where indent/dedent and entry-boundary divergences show
// up. The reverse direction (runtime accepting what syntax junks) is out of
// scope for a public-API-only differential and is exercised by the resolver's
// own unit tests. Intentional forward-direction divergences are listed in the
// allowlists below so unexpected ones still fail loudly.

// droppedMessages lists messages the syntax parser accepts but the runtime
// parser intentionally drops entirely: fixture base name -> message id ->
// reason.
var droppedMessages = map[string]map[string]string{}

// droppedAttributes lists attributes the syntax parser accepts but the runtime
// parser intentionally drops from an otherwise-exposed message: fixture base
// name -> message id -> attribute name -> reason.
var droppedAttributes = map[string]map[string]map[string]string{}

func referenceDropReason(fixture, id string) (string, bool) {
	r, ok := droppedMessages[fixture][id]
	return r, ok
}

func attributeDropReason(fixture, id, attr string) (string, bool) {
	r, ok := droppedAttributes[fixture][id][attr]
	return r, ok
}

// referencePattern renders the expected string for a syntax pattern using only
// text elements and simple placeables whose rendering is independent of the
// resolver and the CLDR formatters. It returns ok=false for any pattern that
// contains a placeable whose value depends on resolution (message/term/function
// references, select expressions, number literals, nested placeables) so the
// caller skips the pattern-level comparison for it.
//
// A string literal renders to its unescaped value via StringLiteral.Parse. An
// unresolved variable reference renders to the fluent.js fallback form {$name}.
func referencePattern(p *ast.Pattern) (string, bool) {
	if p == nil {
		return "", false
	}
	var sb strings.Builder
	for _, el := range p.Elements {
		switch e := el.(type) {
		case *ast.TextElement:
			sb.WriteString(e.Value)
		case *ast.Placeable:
			s, ok := referencePlaceable(e.Expression)
			if !ok {
				return "", false
			}
			sb.WriteString(s)
		default:
			return "", false
		}
	}
	return sb.String(), true
}

func referencePlaceable(expr ast.Expression) (string, bool) {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Parse(), true
	case *ast.VariableReference:
		return "{$" + e.ID.Name + "}", true
	default:
		return "", false
	}
}

func attributeKeys(m map[string]fluent.Pattern) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestRuntimeParserDifferential feeds each reference fixture through both
// parsers and cross-checks the runtime parser against the syntax AST.
func TestRuntimeParserDifferential(t *testing.T) {
	names := ftlFixtures(t, referenceDir)
	require.NotEmptyf(t, names, "no reference fixtures found in %s", referenceDir)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ftlBytes, err := os.ReadFile(filepath.Join(referenceDir, name+".ftl"))
			require.NoErrorf(t, err, "read %s.ftl", name)
			source := string(ftlBytes)

			astRes := syntax.Parse(source, syntax.WithSpans(false))

			bundle := fluent.NewBundle("en", fluent.WithUseIsolating(false))
			bundle.AddResourceOverriding(fluent.NewResource(source))

			// Collect the syntax parser's messages, last-wins on duplicate ids
			// (matching AddResourceOverriding), preserving first-seen order.
			synMsgs := map[string]*ast.Message{}
			var order []string
			for _, e := range astRes.Body {
				m, ok := e.(*ast.Message)
				if !ok {
					continue
				}
				id := m.ID.Name
				if _, seen := synMsgs[id]; !seen {
					order = append(order, id)
				}
				synMsgs[id] = m
			}

			for _, id := range order {
				m := synMsgs[id]
				rm, exposed := bundle.Message(id)

				if reason, dropped := referenceDropReason(name, id); dropped {
					assert.Falsef(t, exposed,
						"message %q is allowlisted as dropped (%s) but the runtime parser exposed it",
						id, reason)
					continue
				}

				require.Truef(t, exposed,
					"runtime parser dropped message %q that the syntax parser accepted", id)

				assertAttributeInventory(t, name, id, m, rm)
				assertPatternText(t, id, "value", m.Value, rm.Value, bundle)

				for _, a := range m.Attributes {
					attr := a.ID.Name
					if _, dropped := attributeDropReason(name, id, attr); dropped {
						continue
					}
					rp, ok := rm.Attributes[attr]
					if !ok {
						continue // reported by assertAttributeInventory
					}
					assertPatternText(t, id, "attribute "+attr, a.Value, rp, bundle)
				}
			}
		})
	}
}

func assertAttributeInventory(t *testing.T, fixture, id string, m *ast.Message, rm *fluent.Message) {
	t.Helper()
	synAttrs := map[string]bool{}
	for _, a := range m.Attributes {
		synAttrs[a.ID.Name] = true
	}

	for attr := range synAttrs {
		if _, dropped := attributeDropReason(fixture, id, attr); dropped {
			continue
		}
		_, ok := rm.Attributes[attr]
		assert.Truef(t, ok,
			"runtime parser dropped attribute %q of message %q that the syntax parser accepted",
			attr, id)
	}

	for _, attr := range attributeKeys(rm.Attributes) {
		assert.Truef(t, synAttrs[attr],
			"runtime parser exposed attribute %q of message %q that the syntax parser did not accept",
			attr, id)
	}
}

func assertPatternText(t *testing.T, id, label string, synPat *ast.Pattern, runPat fluent.Pattern, bundle *fluent.Bundle) {
	t.Helper()
	ref, ok := referencePattern(synPat)
	if !ok {
		return
	}
	got, _ := bundle.FormatPattern(runPat, nil)
	assert.Equalf(t, ref, got,
		"pattern text mismatch for message %q %s", id, label)
}
