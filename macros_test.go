package fluent

import "testing"

// Ported from macros_test.js (term references, parameterization, attributes).

func TestTermReferencesAndCalls(t *testing.T) {
	src := "-bar = Bar\nterm-ref = {-bar}\nterm-call = {-bar()}\n"
	b := newTestBundle(t, src)

	for _, id := range []string{"term-ref", "term-call"} {
		got, errs := format(t, b, id, map[string]any{})
		if got != "Bar" || len(errs) != 0 {
			t.Errorf("%s: got %q errs %v", id, got, errs)
		}
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
		id   string
		args map[string]any
		want string
	}{
		{"ref-foo", map[string]any{}, "Foo {$arg}"},
		{"ref-foo", map[string]any{"arg": 1}, "Foo {$arg}"},
		{"call-foo-no-args", map[string]any{}, "Foo {$arg}"},
		{"call-foo-no-args", map[string]any{"arg": 1}, "Foo {$arg}"},
		{"call-foo-with-expected-arg", map[string]any{}, "Foo 1"},
		{"call-foo-with-expected-arg", map[string]any{"arg": 5}, "Foo 1"},
		{"call-foo-with-other-arg", map[string]any{}, "Foo {$arg}"},
		{"call-foo-with-other-arg", map[string]any{"arg": 5}, "Foo {$arg}"},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, tc.args)
		if got != tc.want || len(errs) != 0 {
			t.Errorf("%s %v: got %q want %q errs %v", tc.id, tc.args, got, tc.want, errs)
		}
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
		id   string
		args map[string]any
		want string
	}{
		{"ref-attr", map[string]any{}, "It"},
		{"ref-attr", map[string]any{"style": "chicago"}, "It"},
		{"call-attr-no-args", map[string]any{}, "It"},
		{"call-attr-no-args", map[string]any{"style": "chicago"}, "It"},
		{"call-attr-with-expected-arg", map[string]any{}, "She"},
		{"call-attr-with-expected-arg", map[string]any{"style": "chicago"}, "She"},
		{"call-attr-with-other-arg", map[string]any{}, "It"},
		{"call-attr-with-other-arg", map[string]any{"style": "chicago"}, "It"},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, tc.args)
		if got != tc.want || len(errs) != 0 {
			t.Errorf("%s %v: got %q want %q errs %v", tc.id, tc.args, got, tc.want, errs)
		}
	}
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
		id   string
		args map[string]any
		want string
	}{
		{"ref-qux", map[string]any{}, "Foo 1"},
		{"ref-qux", map[string]any{"arg": 5}, "Foo 1"},
		{"call-qux-no-args", map[string]any{}, "Foo 1"},
		{"call-qux-with-other", map[string]any{"arg": 5}, "Foo 1"},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, tc.args)
		if got != tc.want || len(errs) != 0 {
			t.Errorf("%s %v: got %q want %q errs %v", tc.id, tc.args, got, tc.want, errs)
		}
	}
}
