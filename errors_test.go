package fluent

import "testing"

// Ported from errors_test.js and bomb_test.js.

func TestErrorReportingIntoArray(t *testing.T) {
	b := newTestBundle(t, "foo = {$one} and {$two}\n")
	msg, _ := b.GetMessage("foo")

	var errs []error
	val := b.FormatPattern(msg.Value, map[string]Value{}, &errs)
	if val != "{$one} and {$two}" {
		t.Errorf("val1 got %q", val)
	}
	if len(errs) != 2 || !isReferenceError(errs[0]) || !isReferenceError(errs[1]) {
		t.Fatalf("expected 2 reference errors, got %v", errs)
	}

	val = b.FormatPattern(msg.Value, map[string]Value{}, &errs)
	if val != "{$one} and {$two}" {
		t.Errorf("val2 got %q", val)
	}
	if len(errs) != 4 {
		t.Errorf("expected 4 accumulated errors, got %d", len(errs))
	}
}

func TestFirstErrorIsThrown(t *testing.T) {
	b := newTestBundle(t, "foo = {$one} and {$two}\n")
	msg, _ := b.GetMessage("foo")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when errs is nil")
		}
		fp, ok := r.(fluentPanic)
		if !ok || !isReferenceError(fp.err) {
			t.Fatalf("expected reference error panic, got %v", r)
		}
	}()
	b.FormatPattern(msg.Value, nil, nil)
}

func TestBillionLaughs(t *testing.T) {
	src := "lol0 = LOL\n" +
		"lol1 = {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0}\n" +
		"lol2 = {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1}\n" +
		"lol3 = {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2}\n" +
		"lol4 = {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3} {lol3}\n" +
		"lol5 = {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4} {lol4}\n" +
		"lol6 = {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5} {lol5}\n" +
		"lol7 = {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6} {lol6}\n" +
		"lol8 = {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7} {lol7}\n" +
		"lol9 = {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8} {lol8}\n" +
		"lolz = {lol9}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "lolz", nil)
	if got != "{???}" {
		t.Errorf("got %q want {???}", got)
	}
	if len(errs) != 1 || !isRangeError(errs[0]) {
		t.Errorf("expected 1 range error, got %v", errs)
	}
}

func TestBillionLaughsThrowsWithoutSink(t *testing.T) {
	src := "lol0 = LOL\n" +
		"lol1 = {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0} {lol0}\n" +
		"lol2 = {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1} {lol1}\n" +
		"lol3 = {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2} {lol2}\n" +
		"lolz = {lol3}\n"
	b := newTestBundle(t, src)
	msg, _ := b.GetMessage("lolz")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		fp, ok := r.(fluentPanic)
		if !ok || !isRangeError(fp.err) {
			t.Fatalf("expected range error panic, got %v", r)
		}
	}()
	b.FormatPattern(msg.Value, nil, nil)
}
