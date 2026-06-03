// Package conformance contains a conformance test suite that proves the Go
// syntax package matches the upstream Project Fluent reference fixtures.
//
// The fixtures under testdata/ are vendored copies of the @fluent/syntax
// reference fixtures (fixtures_structure and fixtures_reference). The
// comparison semantics here mirror the upstream JS suites:
//
//   - structure_test.js: parse with spans ON and deep-compare the AST JSON to
//     the paired .json fixture.
//   - reference_test.js: parse with spans OFF, blank out the annotations array
//     of every Junk entry, then deep-compare.
//
// See structure_test.js, reference_test.js and util.js in the upstream
// @fluent/syntax test directory for the exact behavior being replicated.
package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hakastein/gofluent/syntax"
	"github.com/hakastein/gofluent/syntax/ast"
)

const (
	structureDir = "testdata/structure"
	referenceDir = "testdata/reference"
)

// referenceSkips mirrors the `skips` list in reference_test.js. These fixtures
// produce a different AST in the tooling parser than in the reference parser
// and are skipped by the upstream suite as well.
//
// "leading_dots.ftl": Broken Attributes break the entire Entry right now.
// https://github.com/projectfluent/fluent.js/issues/237
var referenceSkips = map[string]bool{
	"leading_dots.ftl": true,
}

// ftlFixtures returns the sorted list of *.ftl base names (without extension)
// found in dir.
func ftlFixtures(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures dir %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".ftl") {
			names = append(names, strings.TrimSuffix(name, ".ftl"))
		}
	}
	sort.Strings(names)
	return names
}

// readFixture reads the .ftl source and the paired .json expectation for the
// given base name in dir.
func readFixture(t *testing.T, dir, name string) (ftl string, expected []byte) {
	t.Helper()
	ftlBytes, err := os.ReadFile(filepath.Join(dir, name+".ftl"))
	if err != nil {
		t.Fatalf("read %s.ftl: %v", name, err)
	}
	expected, err = os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		t.Fatalf("read %s.json: %v", name, err)
	}
	return string(ftlBytes), expected
}

// toAny unmarshals JSON bytes into an interface{} tree so two payloads can be
// compared semantically (key order independent).
func toAny(t *testing.T, label string, data []byte) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal %s: %v", label, err)
	}
	return v
}

// firstDiff walks two decoded JSON trees and returns a human-readable path to
// the first structural difference, or "" if they are deeply equal.
func firstDiff(path string, got, want interface{}) string {
	if reflect.DeepEqual(got, want) {
		return ""
	}
	switch w := want.(type) {
	case map[string]interface{}:
		g, ok := got.(map[string]interface{})
		if !ok {
			return path + ": type mismatch (got non-object)"
		}
		// Report missing/extra keys first.
		for k := range w {
			if _, ok := g[k]; !ok {
				return path + "." + k + ": missing in got"
			}
		}
		for k := range g {
			if _, ok := w[k]; !ok {
				return path + "." + k + ": extra in got"
			}
		}
		for k, wv := range w {
			if d := firstDiff(path+"."+k, g[k], wv); d != "" {
				return d
			}
		}
	case []interface{}:
		g, ok := got.([]interface{})
		if !ok {
			return path + ": type mismatch (got non-array)"
		}
		if len(g) != len(w) {
			return path + ": length mismatch (got " +
				strconv.Itoa(len(g)) + ", want " + strconv.Itoa(len(w)) + ")"
		}
		for i := range w {
			if d := firstDiff(path+"["+strconv.Itoa(i)+"]", g[i], w[i]); d != "" {
				return d
			}
		}
	}
	return path + ": value mismatch"
}

// assertDeepEqualJSON compares two decoded JSON trees, failing with a readable
// diff (first differing path plus both blobs) on mismatch.
func assertDeepEqualJSON(t *testing.T, name string, gotBytes, wantBytes []byte) {
	t.Helper()
	got := toAny(t, "got", gotBytes)
	want := toAny(t, "want", wantBytes)
	if reflect.DeepEqual(got, want) {
		return
	}
	diff := firstDiff("$", got, want)
	gotPretty, _ := json.MarshalIndent(got, "", "  ")
	wantPretty, _ := json.MarshalIndent(want, "", "  ")
	t.Errorf("AST mismatch for %s\nfirst diff at %s\n--- got ---\n%s\n--- want ---\n%s",
		name, diff, gotPretty, wantPretty)
}

// TestStructureFixtures mirrors structure_test.js: parse with spans ON and
// deep-compare the marshaled AST JSON to the paired fixture.
func TestStructureFixtures(t *testing.T) {
	names := ftlFixtures(t, structureDir)
	if len(names) == 0 {
		t.Fatalf("no structure fixtures found in %s", structureDir)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ftl, expected := readFixture(t, structureDir, name)

			res := syntax.Parse(ftl) // spans ON by default
			got, err := ast.Marshal(res, true)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			assertDeepEqualJSON(t, name, got, expected)
		})
	}
}

// blankJunkAnnotations mirrors the transform in reference_test.js: every Junk
// entry in the resource body has its `annotations` array replaced with []. The
// reference parser does not populate Junk annotations, so they are ignored.
//
// The JS suite applies this only to the parsed side because the fixtures already
// carry empty annotation arrays; applying it to both sides is equivalent and
// guards against the (unlikely) case of a fixture being regenerated with
// populated annotations.
func blankJunkAnnotations(v interface{}) {
	root, ok := v.(map[string]interface{})
	if !ok {
		return
	}
	body, ok := root["body"].([]interface{})
	if !ok {
		return
	}
	for _, e := range body {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if entry["type"] == "Junk" {
			entry["annotations"] = []interface{}{}
		}
	}
}

// TestReferenceFixtures mirrors reference_test.js: parse with spans OFF, blank
// out Junk annotations on both sides, honor the skip-list, then deep-compare.
func TestReferenceFixtures(t *testing.T) {
	names := ftlFixtures(t, referenceDir)
	if len(names) == 0 {
		t.Fatalf("no reference fixtures found in %s", referenceDir)
	}
	for _, name := range names {
		filename := name + ".ftl"
		t.Run(name, func(t *testing.T) {
			if referenceSkips[filename] {
				t.Skipf("skipped to match reference_test.js skip-list (%s)", filename)
			}
			ftl, expected := readFixture(t, referenceDir, name)

			res := syntax.Parse(ftl, syntax.WithSpans(false))
			gotBytes, err := ast.Marshal(res, false)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			got := toAny(t, "got", gotBytes)
			want := toAny(t, "want", expected)
			blankJunkAnnotations(got)
			blankJunkAnnotations(want)

			if reflect.DeepEqual(got, want) {
				return
			}
			diff := firstDiff("$", got, want)
			gotPretty, _ := json.MarshalIndent(got, "", "  ")
			wantPretty, _ := json.MarshalIndent(want, "", "  ")
			t.Errorf("AST mismatch for %s\nfirst diff at %s\n--- got ---\n%s\n--- want ---\n%s",
				name, diff, gotPretty, wantPretty)
		})
	}
}

// roundTripExact holds canonical FTL inputs that the serializer reproduces
// byte-for-byte through Parse + Serialize. These are ported directly from the
// `pretty(input) === input` cases in serializer_test.js, which is where the
// upstream suite actually asserts exact round-trip stability. The structure
// fixtures are intentionally NOT used here: most of them contain Junk, error
// recovery, leading blank lines, or non-canonical placeable spacing, and the
// upstream serializer suite does not round-trip those either.
var roundTripExact = []struct {
	name  string
	input string
}{
	{"simple_message", "foo = Foo\n"},
	{"two_simple_messages", "foo = Foo\nbar = Bar\n"},
	{"simple_term", "-foo = Foo\n"},
	{"block_multiline_message", "foo =\n    Foo\n    Bar\n"},
	{"message_reference", "foo = Foo { bar }\n"},
	{"term_reference", "foo = Foo { -bar }\n"},
	{"external_argument", "foo = Foo { $bar }\n"},
	{"number_element", "foo = Foo { 1 }\n"},
	{"string_element", "foo = Foo { \"bar\" }\n"},
	{"attribute_expression", "foo = Foo { bar.baz }\n"},
	{"resource_comment", "### A multiline\n### resource comment.\n\nfoo = Foo\n"},
	{"message_comment", "# A multiline\n# message comment.\nfoo = Foo\n"},
	{"group_comment", "foo = Foo\n\n## Comment Header\n##\n## A multiline\n## group comment.\n\nbar = Bar\n"},
	{"standalone_comment", "foo = Foo\n\n# A Standalone Comment\n\nbar = Bar\n"},
	{"multiline_with_placeable", "foo =\n    Foo { bar }\n    Baz\n"},
	{"attribute", "foo =\n    .attr = Foo Attr\n"},
	{"two_attributes", "foo =\n    .attr-a = Foo Attr A\n    .attr-b = Foo Attr B\n"},
	{"value_and_attributes", "foo = Foo Value\n    .attr-a = Foo Attr A\n    .attr-b = Foo Attr B\n"},
	{"select_expression", "foo =\n    { $sel ->\n       *[a] A\n        [b] B\n    }\n"},
	{"variant_key_number", "foo =\n    { $sel ->\n       *[1] 1\n    }\n"},
	{"nested_select_expression", "foo =\n    { $a ->\n       *[a]\n            { $b ->\n               *[b] Foo\n            }\n    }\n"},
	{"call_expression", "foo = { FOO() }\n"},
	{"call_expression_positional_and_named", "foo = { FOO(bar, 1, baz: \"baz\") }\n"},
	{"macro_call", "foo = { -term() }\n"},
	{"nested_placeables", "foo = {{ FOO() }}\n"},
	{"backslash_in_text_element", "foo = \\{ placeable }\n"},
	{"escaped_special_char_in_string_literal", "foo = { \"Escaped \\\" quote\" }\n"},
	{"unicode_escape_sequence", "foo = { \"\\u0065\" }\n"},
}

// TestSerializerRoundtripExact asserts byte-for-byte round-trip stability for
// the canonical inputs the upstream serializer_test.js round-trips.
func TestSerializerRoundtripExact(t *testing.T) {
	for _, tc := range roundTripExact {
		t.Run(tc.name, func(t *testing.T) {
			got := syntax.Serialize(syntax.Parse(tc.input))
			if got != tc.input {
				t.Errorf("round-trip mismatch for %s\n--- got ---\n%q\n--- want ---\n%q",
					tc.name, got, tc.input)
			}
		})
	}
}

// TestSerializerRoundtripIdempotent asserts serializer stability across the full
// structure fixture set: serializing a resource and re-parsing it must yield the
// same canonical text. This proves the serializer output is a fixed point
// without requiring the (non-canonical) fixture sources to round-trip exactly.
// Junk is dropped (WithJunk default false), matching serializer_test.js.
func TestSerializerRoundtripIdempotent(t *testing.T) {
	names := ftlFixtures(t, structureDir)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(structureDir, name+".ftl")
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			once := syntax.Serialize(syntax.Parse(string(src)))
			twice := syntax.Serialize(syntax.Parse(once))
			if once != twice {
				t.Errorf("serializer not idempotent for %s\n--- once ---\n%q\n--- twice ---\n%q",
					name, once, twice)
			}
		})
	}
}
