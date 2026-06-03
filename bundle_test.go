package fluent

import (
	"errors"
	"testing"
)

// mustParse parses FTL and fails the test on a hard parse error.
func mustParse(t *testing.T, src string) *Resource {
	t.Helper()
	res, errs := NewResource(src)
	if len(errs) > 0 {
		t.Fatalf("NewResource returned errors: %v", errs)
	}
	return res
}

// newTestBundle creates a bundle with useIsolating=false and adds the resource.
func newTestBundle(t *testing.T, src string) *Bundle {
	t.Helper()
	b := NewBundle("en-US", WithUseIsolating(false))
	b.AddResource(mustParse(t, src))
	return b
}

// format is a convenience helper: get a message value and format it.
func format(t *testing.T, b *Bundle, id string, args map[string]any) (string, []error) {
	t.Helper()
	msg, ok := b.GetMessage(id)
	if !ok {
		t.Fatalf("message %q not found", id)
	}
	var errs []error
	val := b.FormatPatternAny(msg.Value, args, &errs)
	return val, errs
}

func TestAddResource(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	if !b.HasMessage("foo") {
		t.Error("expected message foo")
	}
	if _, ok := b.terms["foo"]; ok {
		t.Error("foo should not be a term")
	}
	if b.HasMessage("-bar") {
		t.Error("-bar should not be a message")
	}
	if _, ok := b.terms["-bar"]; !ok {
		t.Error("expected term -bar")
	}
}

func TestMessagesAndTermsShareName(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	b.AddResource(mustParse(t, "-foo = Private Foo\n"))
	if !b.HasMessage("foo") {
		t.Error("foo should remain a message")
	}
	if _, ok := b.terms["-foo"]; !ok {
		t.Error("expected term -foo")
	}
}

func TestAllowOverrides(t *testing.T) {
	b := NewBundle("en-US", WithUseIsolating(false))
	b.AddResource(mustParse(t, "key = Foo"))

	errs := b.AddResource(mustParse(t, "key = Bar"))
	if len(errs) != 1 {
		t.Fatalf("expected 1 override error, got %d", len(errs))
	}
	msg, _ := b.GetMessage("key")
	if got := b.FormatPattern(msg.Value, nil, nil); got != "Foo" {
		t.Errorf("expected Foo, got %q", got)
	}

	errs = b.AddResourceOverriding(mustParse(t, "key = Bar"))
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors with overriding, got %d", len(errs))
	}
	msg, _ = b.GetMessage("key")
	if got := b.FormatPattern(msg.Value, nil, nil); got != "Bar" {
		t.Errorf("expected Bar, got %q", got)
	}
}

func TestHasMessageBrokenEntries(t *testing.T) {
	src := "foo = Foo\n" +
		"bar =\n" +
		"    .attr = Bar Attr\n" +
		"-term = Term\n" +
		"\n" +
		"err1 =\n" +
		"err2 = {}\n" +
		"err3 =\n" +
		"    .attr =\n" +
		"err4 =\n" +
		"    .attr1 = Attr\n" +
		"    .attr2 = {}\n"
	b := newTestBundle(t, src)

	if !b.HasMessage("foo") {
		t.Error("foo should exist")
	}
	for _, id := range []string{"-term", "missing", "-missing", "err1", "err2", "err3", "err4"} {
		if b.HasMessage(id) {
			t.Errorf("%q should not be a public message", id)
		}
	}
}

func TestGetMessageReturnsValue(t *testing.T) {
	b := newTestBundle(t, "foo = Foo\n-bar = Bar\n")
	msg, ok := b.GetMessage("foo")
	if !ok {
		t.Fatal("expected foo")
	}
	if msg.ID != "foo" || msg.Value != "Foo" || len(msg.Attributes) != 0 {
		t.Errorf("unexpected message: %+v", msg)
	}
	if _, ok := b.GetMessage("-bar"); ok {
		t.Error("-bar should not be retrievable as a message")
	}
}

func isReferenceError(e error) bool {
	var re *referenceError
	return errors.As(e, &re)
}

func isRangeError(e error) bool {
	var re *rangeError
	return errors.As(e, &re)
}

func isTypeError(e error) bool {
	var te *typeError
	return errors.As(e, &te)
}
