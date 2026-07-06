// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"regexp"
	"testing"
)

// firstMessage runs one validator through a fresh Errors and returns the first
// full message (or "" if none), for compact table-driven assertions.
func firstFull(e *Errors) string {
	if e.Empty() {
		return ""
	}
	return e.FullMessages()[0]
}

func TestToFloat(t *testing.T) {
	for _, v := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint64(1), float32(1), float64(1)} {
		if f, ok := toFloat(v); !ok || f != 1 {
			t.Errorf("toFloat(%T) = %v, %v", v, f, ok)
		}
	}
	if _, ok := toFloat("x"); ok {
		t.Error("toFloat string should fail")
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny([]any{"a", "b"}, "b") {
		t.Error("containsAny miss")
	}
	if containsAny([]any{"a"}, "z") {
		t.Error("containsAny false positive")
	}
}

func TestNumRangeCover(t *testing.T) {
	r := NumRange{Min: 1, Max: 10}
	if !r.cover(5) || r.cover(0) || !r.cover(10) {
		t.Error("inclusive range cover wrong")
	}
	ex := NumRange{Min: 1, Max: 10, ExcludeEnd: true}
	if ex.cover(10) || !ex.cover(9) {
		t.Error("exclusive range cover wrong")
	}
	if r.cover("notnum") {
		t.Error("non-numeric should not be covered")
	}
}

func TestPresenceAbsence(t *testing.T) {
	e := newErrors(newModel())
	validatePresence(e, "name", "", nil)
	if firstFull(e) != "Name can't be blank" {
		t.Errorf("presence = %q", firstFull(e))
	}
	e2 := newErrors(newModel())
	validatePresence(e2, "name", "x", nil)
	if !e2.Empty() {
		t.Error("presence ok should be empty")
	}
	e3 := newErrors(newModel())
	validateAbsence(e3, "name", "x", nil)
	if firstFull(e3) != "Name must be blank" {
		t.Errorf("absence = %q", firstFull(e3))
	}
	e4 := newErrors(newModel())
	validateAbsence(e4, "name", "", nil)
	if !e4.Empty() {
		t.Error("absence ok should be empty")
	}
}

func TestAcceptance(t *testing.T) {
	e := newErrors(newModel())
	validateAcceptance(e, "tos", "0", &AcceptanceOptions{}, nil)
	if firstFull(e) != "Tos must be accepted" {
		t.Errorf("acceptance = %q", firstFull(e))
	}
	e2 := newErrors(newModel())
	validateAcceptance(e2, "tos", "1", &AcceptanceOptions{}, nil)
	if !e2.Empty() {
		t.Error("acceptance '1' should pass")
	}
	e3 := newErrors(newModel())
	validateAcceptance(e3, "tos", "yes", &AcceptanceOptions{Accept: []any{"yes"}}, nil)
	if !e3.Empty() {
		t.Error("custom accept should pass")
	}
}

func TestConfirmation(t *testing.T) {
	m := newModel().with("password", "a").with("password_confirmation", "b")
	e := newErrors(m)
	validateConfirmation(m, e, "password", "a", &ConfirmationOptions{}, nil)
	if firstFull(e) != "Password confirmation doesn't match Password" {
		t.Errorf("confirmation = %q", firstFull(e))
	}
	// match -> no error
	m2 := newModel().with("password_confirmation", "a")
	e2 := newErrors(m2)
	validateConfirmation(m2, e2, "password", "a", &ConfirmationOptions{}, nil)
	if !e2.Empty() {
		t.Error("matching confirmation should pass")
	}
	// nil confirmation -> skipped
	m3 := newModel()
	e3 := newErrors(m3)
	validateConfirmation(m3, e3, "password", "a", &ConfirmationOptions{}, nil)
	if !e3.Empty() {
		t.Error("nil confirmation should skip")
	}
	// case-insensitive
	m4 := newModel().with("email_confirmation", "A@B")
	e4 := newErrors(m4)
	validateConfirmation(m4, e4, "email", "a@b", &ConfirmationOptions{CaseSensitive: boolp(false)}, nil)
	if !e4.Empty() {
		t.Error("case-insensitive confirmation should pass")
	}
}

func TestValueLength(t *testing.T) {
	if valueLength(nil) != 0 {
		t.Error("nil length")
	}
	if valueLength("héllo") != 5 {
		t.Error("string rune length")
	}
	if valueLength([]any{1, 2, 3}) != 3 {
		t.Error("slice length")
	}
	if valueLength([]int{1, 2}) != 2 {
		t.Error("typed slice length")
	}
	if valueLength(map[string]int{"a": 1}) != 1 {
		t.Error("map length")
	}
	if valueLength(12345) != 5 {
		t.Error("number to_s length")
	}
}

func TestLength(t *testing.T) {
	e := newErrors(newModel())
	validateLength(e, "name", "a", &LengthOptions{Minimum: intp(2), Maximum: intp(5)}, nil)
	if firstFull(e) != "Name is too short (minimum is 2 characters)" {
		t.Errorf("too short = %q", firstFull(e))
	}
	e2 := newErrors(newModel())
	validateLength(e2, "name", "abcdef", &LengthOptions{Maximum: intp(5)}, nil)
	if firstFull(e2) != "Name is too long (maximum is 5 characters)" {
		t.Errorf("too long = %q", firstFull(e2))
	}
	e3 := newErrors(newModel())
	validateLength(e3, "code", "ab", &LengthOptions{Is: intp(1)}, nil)
	if firstFull(e3) != "Code is the wrong length (should be 1 character)" {
		t.Errorf("wrong length = %q", firstFull(e3))
	}
	// In range: below and above
	e4 := newErrors(newModel())
	validateLength(e4, "name", "a", &LengthOptions{In: &Range{Min: 2, Max: 4}}, nil)
	if firstFull(e4) != "Name is too short (minimum is 2 characters)" {
		t.Errorf("in-short = %q", firstFull(e4))
	}
	e5 := newErrors(newModel())
	validateLength(e5, "name", "abcde", &LengthOptions{In: &Range{Min: 2, Max: 4}}, nil)
	if firstFull(e5) != "Name is too long (maximum is 4 characters)" {
		t.Errorf("in-long = %q", firstFull(e5))
	}
	// valid case
	e6 := newErrors(newModel())
	validateLength(e6, "name", "abc", &LengthOptions{Minimum: intp(2), Maximum: intp(5)}, nil)
	if !e6.Empty() {
		t.Error("length ok should be empty")
	}
	// per-key message override
	e7 := newErrors(newModel())
	validateLength(e7, "name", "abcdef", &LengthOptions{Maximum: intp(5), TooLong: "way too long"}, nil)
	if firstFull(e7) != "Name way too long" {
		t.Errorf("custom too_long = %q", firstFull(e7))
	}
}

func TestFormat(t *testing.T) {
	e := newErrors(newModel())
	validateFormat(e, "email", "x", &FormatOptions{With: regexp.MustCompile(`@`)}, nil)
	if firstFull(e) != "Email is invalid" {
		t.Errorf("format with = %q", firstFull(e))
	}
	e2 := newErrors(newModel())
	validateFormat(e2, "email", "a@b", &FormatOptions{With: regexp.MustCompile(`@`)}, nil)
	if !e2.Empty() {
		t.Error("format with match should pass")
	}
	e3 := newErrors(newModel())
	validateFormat(e3, "email", "a@b", &FormatOptions{Without: regexp.MustCompile(`@`)}, nil)
	if firstFull(e3) != "Email is invalid" {
		t.Errorf("format without = %q", firstFull(e3))
	}
	e4 := newErrors(newModel())
	validateFormat(e4, "email", "ab", &FormatOptions{Without: regexp.MustCompile(`@`)}, nil)
	if !e4.Empty() {
		t.Error("format without non-match should pass")
	}
}

func TestInclusionExclusion(t *testing.T) {
	e := newErrors(newModel())
	validateInclusion(e, "size", "xl", &MembershipOptions{In: []any{"s", "m", "l"}}, nil)
	if firstFull(e) != "Size is not included in the list" {
		t.Errorf("inclusion = %q", firstFull(e))
	}
	e2 := newErrors(newModel())
	validateInclusion(e2, "size", "m", &MembershipOptions{In: []any{"s", "m"}}, nil)
	if !e2.Empty() {
		t.Error("inclusion ok")
	}
	e3 := newErrors(newModel())
	validateExclusion(e3, "name", "admin", &MembershipOptions{In: []any{"admin"}}, nil)
	if firstFull(e3) != "Name is reserved" {
		t.Errorf("exclusion = %q", firstFull(e3))
	}
	e4 := newErrors(newModel())
	validateExclusion(e4, "name", "bob", &MembershipOptions{In: []any{"admin"}}, nil)
	if !e4.Empty() {
		t.Error("exclusion ok")
	}
	// range-based inclusion
	e5 := newErrors(newModel())
	validateInclusion(e5, "n", 50, &MembershipOptions{Range: &NumRange{Min: 1, Max: 10}}, nil)
	if firstFull(e5) != "N is not included in the list" {
		t.Errorf("range inclusion = %q", firstFull(e5))
	}
}

func TestParseNumeric(t *testing.T) {
	cases := []struct {
		v     any
		f     float64
		isInt bool
		ok    bool
	}{
		{5, 5, true, true},
		{int64(5), 5, true, true},
		{int32(5), 5, true, true},
		{5.5, 5.5, false, true},
		{float32(2), 2, false, true},
		{"42", 42, true, true},
		{"1.5", 1.5, false, true},
		{"abc", 0, false, false},
		{true, 0, false, false},
	}
	for _, c := range cases {
		f, isInt, ok := parseNumeric(c.v)
		if f != c.f || isInt != c.isInt || ok != c.ok {
			t.Errorf("parseNumeric(%#v) = %v,%v,%v want %v,%v,%v", c.v, f, isInt, ok, c.f, c.isInt, c.ok)
		}
	}
}

func TestResolveBound(t *testing.T) {
	m := newModel()
	m.methods["limit"] = 42
	if resolveBound(m, Symbol("limit")) != 42 {
		t.Error("symbol bound")
	}
	if resolveBound(m, func(Model) any { return 7 }) != 7 {
		t.Error("proc bound")
	}
	if resolveBound(m, 3) != 3 {
		t.Error("literal bound")
	}
}

func TestMessageOverrides(t *testing.T) {
	// presence -> msgOpts non-nil branch
	e := newErrors(newModel())
	validatePresence(e, "name", "", Symbol("empty"))
	if firstFull(e) != "Name can't be empty" {
		t.Errorf("presence msg override = %q", firstFull(e))
	}
	// confirmation -> message branch
	m := newModel().with("password", "a").with("password_confirmation", "b")
	e2 := newErrors(m)
	validateConfirmation(m, e2, "password", "a", &ConfirmationOptions{Message: "mismatch"}, nil)
	if firstFull(e2) != "Password confirmation mismatch" {
		t.Errorf("confirmation msg override = %q", firstFull(e2))
	}
	// inclusion/exclusion -> membershipOpts message branch
	e3 := newErrors(newModel())
	validateInclusion(e3, "size", "xl", &MembershipOptions{In: []any{"s"}, Message: "bad size"}, nil)
	if firstFull(e3) != "Size bad size" {
		t.Errorf("inclusion msg override = %q", firstFull(e3))
	}
	// numericality -> numOpts message branch
	e4 := newErrors(newModel())
	validateNumericality(newModel(), e4, "age", "x", &NumericalityOptions{Message: "not numeric"}, nil)
	if firstFull(e4) != "Age not numeric" {
		t.Errorf("numericality msg override = %q", firstFull(e4))
	}
}

func TestNumericality(t *testing.T) {
	run := func(value any, o *NumericalityOptions) *Errors {
		m := newModel().with("n", value)
		e := newErrors(m)
		validateNumericality(m, e, "n", value, o, nil)
		return e
	}
	if got := firstFull(run("abc", &NumericalityOptions{})); got != "N is not a number" {
		t.Errorf("nan = %q", got)
	}
	if got := firstFull(run("1.5", &NumericalityOptions{OnlyInteger: true})); got != "N must be an integer" {
		t.Errorf("not int = %q", got)
	}
	if got := firstFull(run(10, &NumericalityOptions{GreaterThan: 18})); got != "N must be greater than 18" {
		t.Errorf("gt = %q", got)
	}
	if got := firstFull(run(3, &NumericalityOptions{GreaterThanOrEqualTo: 5})); got != "N must be greater than or equal to 5" {
		t.Errorf("gte = %q", got)
	}
	if got := firstFull(run(4, &NumericalityOptions{EqualTo: 3})); got != "N must be equal to 3" {
		t.Errorf("eq = %q", got)
	}
	if got := firstFull(run(10, &NumericalityOptions{LessThan: 5})); got != "N must be less than 5" {
		t.Errorf("lt = %q", got)
	}
	if got := firstFull(run(10, &NumericalityOptions{LessThanOrEqualTo: 5})); got != "N must be less than or equal to 5" {
		t.Errorf("lte = %q", got)
	}
	if got := firstFull(run(7, &NumericalityOptions{OtherThan: 7})); got != "N must be other than 7" {
		t.Errorf("other = %q", got)
	}
	if got := firstFull(run(4, &NumericalityOptions{Odd: true})); got != "N must be odd" {
		t.Errorf("odd = %q", got)
	}
	if got := firstFull(run(3, &NumericalityOptions{Even: true})); got != "N must be even" {
		t.Errorf("even = %q", got)
	}
	// all pass -> empty
	if e := run(20, &NumericalityOptions{GreaterThan: 1, OnlyInteger: true, Odd: false, Even: true, LessThan: 100, LessThanOrEqualTo: 100, GreaterThanOrEqualTo: 1, EqualTo: 20, OtherThan: 5}); !e.Empty() {
		t.Errorf("all-pass should be empty, got %v", e.FullMessages())
	}
	// two errors, ordering: equal_to before odd
	e := run(4, &NumericalityOptions{EqualTo: 3, Odd: true})
	msgs := e.FullMessages()
	if len(msgs) != 2 || msgs[0] != "N must be equal to 3" || msgs[1] != "N must be odd" {
		t.Errorf("ordering = %#v", msgs)
	}
}
