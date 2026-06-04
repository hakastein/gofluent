package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		t.Run(tc.id, func(t *testing.T) {
			got, errs := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			assert.Len(t, errs, tc.wantErr)
		})
	}

	// Attributes directly.
	msg, _ := b.GetMessage("key5")
	var errs []error
	assert.Equal(t, "A5", b.FormatPattern(msg.Attributes["a"], nil, &errs))
	assert.Equal(t, "B5", b.FormatPattern(msg.Attributes["b"], nil, &errs))
	assert.Empty(t, errs)
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
		t.Run(tc.id, func(t *testing.T) {
			got, errs := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			assert.Len(t, errs, tc.wantErr)
		})
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
		wantErr int
	}{
		{"ref-message", "Foo", 0},
		{"ref-term", "Bar", 0},
		{"ref-missing-message", "{missing}", 1},
		{"ref-missing-term", "{-missing}", 1},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			got, errs := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			require.Len(t, errs, tc.wantErr)
			if tc.wantErr > 0 {
				require.ErrorIs(t, errs[0], fluent.ErrReference, "missing references report a reference error")
			}
		})
	}
}

func TestNullValueReference(t *testing.T) {
	src := "foo =\n" +
		"    .attr = Foo Attr\n" +
		"bar = { foo } Bar\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "{???}", got)
	assert.Len(t, errs, 1)

	msg, _ := b.GetMessage("foo")
	var aerr []error
	assert.Equal(t, "Foo Attr", b.FormatPattern(msg.Attributes["attr"], nil, &aerr))

	got, errs = format(t, b, "bar", nil)
	assert.Equal(t, "{foo} Bar", got)
	require.Len(t, errs, 1, "referencing a value-less message reports a reference error")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
}

func TestCyclicReferences(t *testing.T) {
	t.Run("mutual", func(t *testing.T) {
		b := newTestBundle(t, "foo = { bar }\nbar = { foo }\n")
		got, errs := format(t, b, "foo", nil)
		assert.Equal(t, "{???}", got)
		require.NotEmpty(t, errs, "a cycle reports a range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
	})
	t.Run("self", func(t *testing.T) {
		b := newTestBundle(t, "foo = { foo }\n")
		got, errs := format(t, b, "foo", nil)
		assert.Equal(t, "{???}", got)
		require.NotEmpty(t, errs, "a self-reference reports a range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
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
		assert.Equal(t, "{???}", got)
		require.NotEmpty(t, errs, "a self-reference in the selected member reports a range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)

		got, errs = format(t, b, "foo", map[string]any{"sel": "b"})
		assert.Equal(t, "Bar", got)
		assert.Empty(t, errs)
	})
}
