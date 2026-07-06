// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"reflect"
	"testing"
)

func newErrors(m Model) *Errors { return NewErrors(m, NewName("Person")) }

func TestInterpolate(t *testing.T) {
	got := interpolate("must be %{count} for %{who}", map[string]any{"count": 5})
	if got != "must be 5 for %{who}" {
		t.Errorf("interpolate = %q", got)
	}
}

func TestToInt(t *testing.T) {
	cases := []struct {
		v    any
		want int
	}{
		{3, 3}, {int64(4), 4}, {float64(5), 5}, {"x", 0},
	}
	for _, c := range cases {
		if got := toInt(c.v); got != c.want {
			t.Errorf("toInt(%#v) = %d, want %d", c.v, got, c.want)
		}
	}
}

func TestLookupTemplate(t *testing.T) {
	if got := lookupTemplate("too_short", map[string]any{"count": 1}); got != "is too short (minimum is 1 character)" {
		t.Errorf("one form: %q", got)
	}
	if got := lookupTemplate("too_short", map[string]any{"count": 3}); got != "is too short (minimum is %{count} characters)" {
		t.Errorf("other form: %q", got)
	}
	if got := lookupTemplate("blank", nil); got != "can't be blank" {
		t.Errorf("table: %q", got)
	}
	if got := lookupTemplate("nope", nil); got != "translation missing: en.errors.messages.nope" {
		t.Errorf("missing: %q", got)
	}
}

func TestErrorTypeDefault(t *testing.T) {
	e := newErrors(newModel())
	err := e.Add("name", nil, nil)
	if err.Type() != Symbol("invalid") {
		t.Errorf("default type = %#v", err.Type())
	}
	err2 := e.Add("name", Symbol("blank"), nil)
	if err2.Type() != Symbol("blank") {
		t.Errorf("type = %#v", err2.Type())
	}
}

func TestMessageLiteralVsSymbol(t *testing.T) {
	e := newErrors(newModel())
	lit := e.Add("name", "is totally wrong", nil)
	if lit.Message() != "is totally wrong" {
		t.Errorf("literal string message = %q", lit.Message())
	}
	sym := e.Add("name", Symbol("blank"), nil)
	if sym.Message() != "can't be blank" {
		t.Errorf("symbol message = %q", sym.Message())
	}
	other := e.Add("name", 42, nil)
	if other.Message() != "42" {
		t.Errorf("non-symbol/non-string message = %q", other.Message())
	}
}

func TestGenerateMessageOverrides(t *testing.T) {
	e := newErrors(newModel())
	// Symbol :message overrides the key.
	symOv := e.Add("name", Symbol("blank"), map[string]any{"message": Symbol("empty")})
	if symOv.Message() != "can't be empty" {
		t.Errorf("symbol override = %q", symOv.Message())
	}
	// String :message is an interpolated template.
	strOv := e.Add("name", Symbol("too_short"), map[string]any{"message": "min %{count}", "count": 3})
	if strOv.Message() != "min 3" {
		t.Errorf("string override = %q", strOv.Message())
	}
	// Proc :message.
	procOv := e.Add("name", Symbol("blank"), map[string]any{"message": func(Model) string { return "from proc" }})
	if procOv.Message() != "from proc" {
		t.Errorf("proc override = %q", procOv.Message())
	}
}

func TestAttrValueInterpolation(t *testing.T) {
	m := newModel().with("name", "Bob")
	e := newErrors(m)
	// %{value} pulls from the model via the respond_to? guard.
	err := e.Add("name", Symbol("blank"), map[string]any{"message": "was %{value}"})
	if got := err.Message(); got != "was Bob" {
		t.Errorf("value interp = %q", got)
	}
	// base attribute never reads a value.
	base := e.Add("base", Symbol("blank"), map[string]any{"message": "v=%{value}"})
	if got := base.Message(); got != "v=" {
		t.Errorf("base value interp = %q", got)
	}
	// unknown attribute (not responded to) reads nil.
	unknown := e.Add("missing", Symbol("blank"), map[string]any{"message": "v=%{value}"})
	if got := unknown.Message(); got != "v=" {
		t.Errorf("unknown value interp = %q", got)
	}
}

func TestNormalizeTypeProc(t *testing.T) {
	e := newErrors(newModel())
	err := e.Add("name", func(Model) any { return Symbol("blank") }, nil)
	if err.Type() != Symbol("blank") {
		t.Errorf("proc type = %#v", err.Type())
	}
}

func TestFullMessage(t *testing.T) {
	e := newErrors(newModel())
	e.Add("first_name", Symbol("blank"), nil)
	if got := e.FullMessages()[0]; got != "First name can't be blank" {
		t.Errorf("full message = %q", got)
	}
	e.Add("base", "is broken", nil)
	if got := e.FullMessages()[1]; got != "is broken" {
		t.Errorf("base full message = %q", got)
	}
}

func TestDetails(t *testing.T) {
	e := newErrors(newModel())
	e.Add("name", Symbol("too_short"), map[string]any{"count": 3, "if": "cond", "message": "x"})
	d := e.Details()["name"][0]
	want := map[string]any{"error": Symbol("too_short"), "count": 3}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("details = %#v, want %#v", d, want)
	}
}

func TestWhereAndMatch(t *testing.T) {
	e := newErrors(newModel())
	e.Add("name", Symbol("too_short"), map[string]any{"count": 3})
	e.Add("email", Symbol("blank"), nil)
	if len(e.Where("name", nil, nil)) != 1 {
		t.Error("where by attr")
	}
	if len(e.Where("name", Symbol("too_short"), map[string]any{"count": 3})) != 1 {
		t.Error("where by attr+type+opts")
	}
	if len(e.Where("name", Symbol("too_short"), map[string]any{"count": 9})) != 0 {
		t.Error("where opt mismatch should be empty")
	}
	if len(e.Where("name", Symbol("blank"), nil)) != 0 {
		t.Error("where type mismatch should be empty")
	}
	if len(e.Where("zzz", nil, nil)) != 0 {
		t.Error("where attr mismatch should be empty")
	}
}

func TestAddedAndOfKind(t *testing.T) {
	e := newErrors(newModel())
	e.Add("name", Symbol("too_short"), map[string]any{"count": 3})
	e.Add("email", "custom text", nil)

	if !e.Added("name", Symbol("too_short"), map[string]any{"count": 3}) {
		t.Error("added symbol strict match should be true")
	}
	if e.Added("name", Symbol("too_short"), map[string]any{"count": 9}) {
		t.Error("added strict mismatch should be false")
	}
	if !e.Added("email", "custom text", nil) {
		t.Error("added string message should be true")
	}
	if e.Added("email", "other text", nil) {
		t.Error("added missing string message should be false")
	}
	if !e.OfKind("name", Symbol("too_short")) {
		t.Error("ofKind symbol should be true")
	}
	if e.OfKind("name", Symbol("blank")) {
		t.Error("ofKind wrong symbol should be false")
	}
	if !e.OfKind("email", "custom text") {
		t.Error("ofKind string should be true")
	}
	if e.OfKind("email", "nope") {
		t.Error("ofKind missing string should be false")
	}
}

func TestStrictMatchNilOpts(t *testing.T) {
	e := newErrors(newModel())
	e.Add("name", Symbol("blank"), nil)
	if !e.Added("name", Symbol("blank"), nil) {
		t.Error("added with nil opts and no options should match")
	}
}

func TestCollectionAccessors(t *testing.T) {
	m := newModel()
	e := newErrors(m)
	if !e.Empty() || e.Any() || e.Size() != 0 {
		t.Error("empty errors state wrong")
	}
	e.Add("name", Symbol("blank"), nil)
	e.Add("name", Symbol("invalid"), nil)
	e.Add("email", Symbol("blank"), nil)

	if e.Empty() || !e.Any() || e.Size() != 3 {
		t.Error("populated errors state wrong")
	}
	if !e.Include("name") || e.Include("zzz") {
		t.Error("include wrong")
	}
	if got := e.Get("name"); len(got) != 2 || got[0] != "can't be blank" {
		t.Errorf("get(name) = %#v", got)
	}
	if got := e.MessagesFor("email"); len(got) != 1 {
		t.Errorf("messagesFor = %#v", got)
	}
	if got := e.FullMessagesFor("email"); got[0] != "Email can't be blank" {
		t.Errorf("fullMessagesFor = %#v", got)
	}
	if got := e.Messages()["email"]; len(got) != 1 {
		t.Errorf("messages = %#v", got)
	}
	names := e.AttributeNames()
	if !reflect.DeepEqual(names, []string{"name", "email"}) {
		t.Errorf("attributeNames = %#v", names)
	}
	if len(e.Entries()) != 3 {
		t.Error("entries")
	}
	count := 0
	e.Each(func(*Error) { count++ })
	if count != 3 {
		t.Error("each count")
	}
	first := e.Entries()[0]
	if first.Attribute() != "name" || first.Options() == nil {
		t.Error("error accessors")
	}
	e.Clear()
	if !e.Empty() {
		t.Error("clear")
	}
}

func TestTypeZeroValue(t *testing.T) {
	// A zero Error (never produced by Add, which normalizes) defaults to :invalid.
	if (&Error{}).Type() != Symbol("invalid") {
		t.Error("zero Error type should default to :invalid")
	}
}

func TestStrictMatchSkipsControlOptions(t *testing.T) {
	e := newErrors(newModel())
	e.Add("name", Symbol("too_short"), map[string]any{"count": 3, "if": "cond", "message": "custom"})
	// The callback/message options are ignored by the strict match.
	if !e.Added("name", Symbol("too_short"), map[string]any{"count": 3}) {
		t.Error("strict match should ignore if:/message: options")
	}
}

func TestTypeEqual(t *testing.T) {
	if !typeEqual(Symbol("a"), Symbol("a")) {
		t.Error("symbol equal")
	}
	if typeEqual(Symbol("a"), "a") {
		t.Error("symbol vs string should differ")
	}
	if !typeEqual("a", "a") {
		t.Error("string equal")
	}
}
