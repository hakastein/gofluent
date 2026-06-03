package fluent

import "testing"

// Ported from values_format_test.js and values_ref_test.js and patterns_test.js.

func TestFormattingValues(t *testing.T) {
	src := "key1 = Value 1\n" +
		"key2 = { $sel ->\n" +
		"    [a] A2\n" +
		"   *[b] B2\n" +
		"}\n" +
		"key3 = Value { 3 }\n" +
		"key4 = { $sel ->\n" +
		"    [a] A{ 4 }\n" +
		"   *[b] B{ 4 }\n" +
		"}\n" +
		"key5 =\n" +
		"    .a = A5\n" +
		"    .b = B5\n"
	b := newTestBundle(t, src)

	tests := []struct {
		id      string
		want    string
		wantErr int
	}{
		{"key1", "Value 1", 0},
		{"key2", "B2", 1},
		{"key3", "Value 3", 0},
		{"key4", "B4", 1},
		{"key5", "{???}", 1},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, nil)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.id, got, tc.want)
		}
		if len(errs) != tc.wantErr {
			t.Errorf("%s: got %d errors want %d (%v)", tc.id, len(errs), tc.wantErr, errs)
		}
	}

	// Attributes directly.
	msg, _ := b.GetMessage("key5")
	var errs []error
	if got := b.FormatPattern(msg.Attributes["a"], nil, &errs); got != "A5" {
		t.Errorf("key5.a got %q", got)
	}
	if got := b.FormatPattern(msg.Attributes["b"], nil, &errs); got != "B5" {
		t.Errorf("key5.b got %q", got)
	}
	if len(errs) != 0 {
		t.Errorf("attr format errors: %v", errs)
	}
}

func TestReferencingValues(t *testing.T) {
	src := "key1 = Value 1\n" +
		"-key2 = { $sel ->\n" +
		"    [a] A2\n" +
		"   *[b] B2\n" +
		"}\n" +
		"key3 = Value { 3 }\n" +
		"-key4 = { $sel ->\n" +
		"    [a] A{ 4 }\n" +
		"   *[b] B{ 4 }\n" +
		"}\n" +
		"key5 =\n" +
		"    .a = A5\n" +
		"    .b = B5\n" +
		"\n" +
		"ref1 = { key1 }\n" +
		"ref2 = { -key2 }\n" +
		"ref3 = { key3 }\n" +
		"ref4 = { -key4 }\n" +
		"ref5 = { key5 }\n" +
		"\n" +
		"ref6 = { -key2(sel: \"a\") }\n" +
		"ref7 = { -key2(sel: \"b\") }\n" +
		"\n" +
		"ref8 = { -key4(sel: \"a\") }\n" +
		"ref9 = { -key4(sel: \"b\") }\n" +
		"\n" +
		"ref10 = { key5.a }\n" +
		"ref11 = { key5.b }\n" +
		"ref12 = { key5.c }\n" +
		"\n" +
		"ref13 = { key6 }\n" +
		"ref14 = { key6.a }\n" +
		"\n" +
		"ref15 = { -key6 }\n" +
		"ref16 = { -key6.a ->\n" +
		"    *[a] A\n" +
		"}\n"
	b := newTestBundle(t, src)

	tests := []struct {
		id      string
		want    string
		wantErr int
	}{
		{"ref1", "Value 1", 0},
		{"ref2", "B2", 0},
		{"ref3", "Value 3", 0},
		{"ref4", "B4", 0},
		{"ref5", "{key5}", 1},
		{"ref6", "A2", 0},
		{"ref7", "B2", 0},
		{"ref8", "A4", 0},
		{"ref9", "B4", 0},
		{"ref10", "A5", 0},
		{"ref11", "B5", 0},
		{"ref12", "{key5.c}", 1},
		{"ref13", "{key6}", 1},
		{"ref14", "{key6}", 1},
		{"ref15", "{-key6}", 1},
		{"ref16", "A", 1},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, nil)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.id, got, tc.want)
		}
		if len(errs) != tc.wantErr {
			t.Errorf("%s: got %d errors want %d (%v)", tc.id, len(errs), tc.wantErr, errs)
		}
	}
}

func TestPatternsReferences(t *testing.T) {
	src := "foo = Foo\n" +
		"-bar = Bar\n" +
		"\n" +
		"ref-message = { foo }\n" +
		"ref-term = { -bar }\n" +
		"\n" +
		"ref-missing-message = { missing }\n" +
		"ref-missing-term = { -missing }\n"
	b := newTestBundle(t, src)

	tests := []struct {
		id      string
		want    string
		wantErr bool
	}{
		{"ref-message", "Foo", false},
		{"ref-term", "Bar", false},
		{"ref-missing-message", "{missing}", true},
		{"ref-missing-term", "{-missing}", true},
	}
	for _, tc := range tests {
		got, errs := format(t, b, tc.id, nil)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.id, got, tc.want)
		}
		if tc.wantErr && (len(errs) == 0 || !isReferenceError(errs[0])) {
			t.Errorf("%s: expected reference error, got %v", tc.id, errs)
		}
	}
}

func TestNullValueReference(t *testing.T) {
	src := "foo =\n" +
		"    .attr = Foo Attr\n" +
		"bar = { foo } Bar\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "foo", nil)
	if got != "{???}" || len(errs) != 1 {
		t.Errorf("foo: got %q errs %v", got, errs)
	}

	msg, _ := b.GetMessage("foo")
	var aerr []error
	if got := b.FormatPattern(msg.Attributes["attr"], nil, &aerr); got != "Foo Attr" {
		t.Errorf("foo.attr got %q", got)
	}

	got, errs = format(t, b, "bar", nil)
	if got != "{foo} Bar" || len(errs) != 1 || !isReferenceError(errs[0]) {
		t.Errorf("bar: got %q errs %v", got, errs)
	}
}

func TestCyclicReferences(t *testing.T) {
	t.Run("mutual", func(t *testing.T) {
		b := newTestBundle(t, "foo = { bar }\nbar = { foo }\n")
		got, errs := format(t, b, "foo", nil)
		if got != "{???}" || len(errs) == 0 || !isRangeError(errs[0]) {
			t.Errorf("got %q errs %v", got, errs)
		}
	})
	t.Run("self", func(t *testing.T) {
		b := newTestBundle(t, "foo = { foo }\n")
		got, errs := format(t, b, "foo", nil)
		if got != "{???}" || len(errs) == 0 || !isRangeError(errs[0]) {
			t.Errorf("got %q errs %v", got, errs)
		}
	})
	t.Run("self in member", func(t *testing.T) {
		src := "foo =\n" +
			"    { $sel ->\n" +
			"       *[a] { foo }\n" +
			"        [b] Bar\n" +
			"    }\n" +
			"bar = { foo }\n"
		b := newTestBundle(t, src)
		got, errs := format(t, b, "foo", map[string]any{"sel": "a"})
		if got != "{???}" || len(errs) == 0 || !isRangeError(errs[0]) {
			t.Errorf("a: got %q errs %v", got, errs)
		}
		got, errs = format(t, b, "foo", map[string]any{"sel": "b"})
		if got != "Bar" || len(errs) != 0 {
			t.Errorf("b: got %q errs %v", got, errs)
		}
	})
}
