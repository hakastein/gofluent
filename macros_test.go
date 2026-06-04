package fluent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ported from macros_test.js (term references, parameterization, attributes).

func TestTermReferencesAndCalls(t *testing.T) {
	src := "-bar = Bar\nterm-ref = {-bar}\nterm-call = {-bar()}\n"
	b := newTestBundle(t, src)

	for _, id := range []string{"term-ref", "term-call"} {
		got, errs := format(t, b, id, map[string]any{})
		assert.Equalf(t, "Bar", got, "%s", id)
		assert.Emptyf(t, errs, "%s", id)
	}
}

func TestTermPassingArguments(t *testing.T) {
	src := "-foo = Foo {$arg}\n" +
		"\n" +
		"ref-foo = {-foo}\n" +
		"call-foo-no-args = {-foo()}\n" +
		"call-foo-with-expected-arg = {-foo(arg: 1)}\n" +
		"call-foo-with-other-arg = {-foo(other: 3)}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		name string
		id   string
		args map[string]any
		want string
	}{
		{"ref-foo no args", "ref-foo", map[string]any{}, "Foo {$arg}"},
		{"ref-foo with arg", "ref-foo", map[string]any{"arg": 1}, "Foo {$arg}"},
		{"call no args", "call-foo-no-args", map[string]any{}, "Foo {$arg}"},
		{"call no args with caller arg", "call-foo-no-args", map[string]any{"arg": 1}, "Foo {$arg}"},
		{"call expected arg", "call-foo-with-expected-arg", map[string]any{}, "Foo 1"},
		{"call expected arg ignores caller", "call-foo-with-expected-arg", map[string]any{"arg": 5}, "Foo 1"},
		{"call other arg", "call-foo-with-other-arg", map[string]any{}, "Foo {$arg}"},
		{"call other arg ignores caller", "call-foo-with-other-arg", map[string]any{"arg": 5}, "Foo {$arg}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := format(t, b, tc.id, tc.args)
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}
}

func TestParameterizedTermAttributes(t *testing.T) {
	src := "-ship = Ship\n" +
		"    .gender = {$style ->\n" +
		"       *[traditional] neuter\n" +
		"        [chicago] feminine\n" +
		"    }\n" +
		"\n" +
		"ref-attr = {-ship.gender ->\n" +
		"   *[masculine] He\n" +
		"    [feminine] She\n" +
		"    [neuter] It\n" +
		"}\n" +
		"call-attr-no-args = {-ship.gender() ->\n" +
		"   *[masculine] He\n" +
		"    [feminine] She\n" +
		"    [neuter] It\n" +
		"}\n" +
		"call-attr-with-expected-arg = {-ship.gender(style: \"chicago\") ->\n" +
		"   *[masculine] He\n" +
		"    [feminine] She\n" +
		"    [neuter] It\n" +
		"}\n" +
		"call-attr-with-other-arg = {-ship.gender(other: 3) ->\n" +
		"   *[masculine] He\n" +
		"    [feminine] She\n" +
		"    [neuter] It\n" +
		"}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		name string
		id   string
		args map[string]any
		want string
	}{
		{"ref no args", "ref-attr", map[string]any{}, "It"},
		{"ref ignores caller style", "ref-attr", map[string]any{"style": "chicago"}, "It"},
		{"call no args", "call-attr-no-args", map[string]any{}, "It"},
		{"call no args ignores caller style", "call-attr-no-args", map[string]any{"style": "chicago"}, "It"},
		{"call expected arg", "call-attr-with-expected-arg", map[string]any{}, "She"},
		{"call expected arg ignores caller", "call-attr-with-expected-arg", map[string]any{"style": "chicago"}, "She"},
		{"call other arg", "call-attr-with-other-arg", map[string]any{}, "It"},
		{"call other arg ignores caller style", "call-attr-with-other-arg", map[string]any{"style": "chicago"}, "It"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := format(t, b, tc.id, tc.args)
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}
}

// TestTermParamsClearedAfterNestedTerm pins the fluent.js scoping rule: a
// TermReference sets scope.params for the duration of its own body and then
// nulls them out unconditionally (resolver.ts lines 239-251: `scope.params =
// getArguments(...).named` ... `scope.params = null`). It does NOT restore a
// previously-active params bag. So once an outer term has embedded an inner
// term, any variable the outer term references AFTER that embedding is resolved
// against the top-level args, not against the outer term's own params.
//
// Here -outer embeds -inner, then references $x. With the fluent.js null-out
// rule the trailing {$x} sees scope.params == null after -inner resolves and so
// reads the top-level arg "top". (A restore-prev implementation would instead
// surface the outer term's own arg "term".)
func TestTermParamsClearedAfterNestedTerm(t *testing.T) {
	src := "-inner = Inner\n" +
		"-outer = {-inner} {$x}\n" +
		"msg = {-outer(x: \"term\")}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "msg", map[string]any{"x": "top"})
	assert.Equal(t, "Inner top", got)
	assert.Empty(t, errs)
}

func TestNestingTermReferences(t *testing.T) {
	src := "-foo = Foo {$arg}\n" +
		"-bar = {-foo}\n" +
		"-baz = {-foo()}\n" +
		"-qux = {-foo(arg: 1)}\n" +
		"\n" +
		"ref-qux = {-qux}\n" +
		"call-qux-no-args = {-qux()}\n" +
		"call-qux-with-other = {-qux(other: 3)}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		name string
		id   string
		args map[string]any
		want string
	}{
		{"ref no args", "ref-qux", map[string]any{}, "Foo 1"},
		{"ref ignores caller arg", "ref-qux", map[string]any{"arg": 5}, "Foo 1"},
		{"call no args", "call-qux-no-args", map[string]any{}, "Foo 1"},
		{"call other arg ignores caller", "call-qux-with-other", map[string]any{"arg": 5}, "Foo 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := format(t, b, tc.id, tc.args)
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}
}
