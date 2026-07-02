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
		wantErr error // nil means no errors expected
	}{
		{"key1", "Value 1", nil},
		{"key2", "B2", fluent.ErrReference},
		{"key3", "Value 3", nil},
		{"key4", "B4", fluent.ErrReference},
		{"key5", "{???}", fluent.ErrType},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			got, err := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}

	// Attributes directly.
	msg, _ := b.Message("key5")
	got, err := b.FormatPattern(msg.Attributes["a"], nil)
	assert.Equal(t, "A5", got)
	assert.NoError(t, err)
	got, err = b.FormatPattern(msg.Attributes["b"], nil)
	assert.Equal(t, "B5", got)
	assert.NoError(t, err)
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
		wantErr error // nil means no errors expected
	}{
		{"ref1", "Value 1", nil},
		{"ref2", "B2", nil},
		{"ref3", "Value 3", nil},
		{"ref4", "B4", nil},
		{"ref5", "{key5}", fluent.ErrReference},
		{"ref6", "A2", nil},
		{"ref7", "B2", nil},
		{"ref8", "A4", nil},
		{"ref9", "B4", nil},
		{"ref10", "A5", nil},
		{"ref11", "B5", nil},
		{"ref12", "{key5.c}", fluent.ErrReference},
		{"ref13", "{key6}", fluent.ErrReference},
		{"ref14", "{key6}", fluent.ErrReference},
		{"ref15", "{-key6}", fluent.ErrReference},
		{"ref16", "A", fluent.ErrReference},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			got, err := format(t, b, tc.id, nil)
			assert.Equal(t, tc.want, got)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestNullValueReference(t *testing.T) {
	src := "foo =\n" +
		"    .attr = Foo Attr\n" +
		"bar = { foo } Bar\n"
	b := newTestBundle(t, src)

	got, err := format(t, b, "foo", nil)
	assert.Equal(t, "{???}", got)
	require.ErrorIs(t, err, fluent.ErrType)

	msg, _ := b.Message("foo")
	got, err = b.FormatPattern(msg.Attributes["attr"], nil)
	assert.Equal(t, "Foo Attr", got)
	assert.NoError(t, err)

	got, err = format(t, b, "bar", nil)
	assert.Equal(t, "{foo} Bar", got)
	require.ErrorIs(t, err, fluent.ErrReference, "referencing a value-less message reports a reference error")
}

func TestCyclicReferences(t *testing.T) {
	t.Run("mutual", func(t *testing.T) {
		b := newTestBundle(t, "foo = { bar }\nbar = { foo }\n")
		got, err := format(t, b, "foo", nil)
		assert.Equal(t, "{???}", got)
		require.ErrorIs(t, err, fluent.ErrRange, "a cycle reports a range error")
	})
	t.Run("self", func(t *testing.T) {
		b := newTestBundle(t, "foo = { foo }\n")
		got, err := format(t, b, "foo", nil)
		assert.Equal(t, "{???}", got)
		require.ErrorIs(t, err, fluent.ErrRange, "a self-reference reports a range error")
	})
	t.Run("self in member", func(t *testing.T) {
		src := "foo =\n" +
			"    { $sel ->\n" +
			"       *[a] { foo }\n" +
			"        [b] Bar\n" +
			"    }\n" +
			"bar = { foo }\n"
		b := newTestBundle(t, src)

		got, err := format(t, b, "foo", map[string]any{"sel": "a"})
		assert.Equal(t, "{???}", got)
		require.ErrorIs(t, err, fluent.ErrRange, "a self-reference in the selected member reports a range error")

		got, err = format(t, b, "foo", map[string]any{"sel": "b"})
		assert.Equal(t, "Bar", got)
		assert.NoError(t, err)
	})
}
