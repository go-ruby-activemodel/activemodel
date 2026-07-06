// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Range is an inclusive integer range used by length in:/within:.
type Range struct{ Min, Max int }

// NumRange is a numeric range used by inclusion/exclusion in: a Ruby Range. When
// ExcludeEnd is set it models the "..." (end-exclusive) form.
type NumRange struct {
	Min, Max   float64
	ExcludeEnd bool
}

func (r NumRange) cover(value any) bool {
	f, ok := toFloat(value)
	if !ok {
		return false
	}
	if f < r.Min {
		return false
	}
	if r.ExcludeEnd {
		return f < r.Max
	}
	return f <= r.Max
}

// LengthOptions configures the length validator (validates length:). Minimum,
// Maximum and Is are optional; In is the range form. The per-key message
// overrides (TooLong/TooShort/WrongLength) and Message mirror Rails.
type LengthOptions struct {
	Minimum     *int
	Maximum     *int
	Is          *int
	In          *Range
	Message     any
	TooLong     any
	TooShort    any
	WrongLength any
}

// FormatOptions configures the format validator (validates format:). With must
// match; Without must not.
type FormatOptions struct {
	With    *regexp.Regexp
	Without *regexp.Regexp
	Message any
}

// MembershipOptions configures inclusion/exclusion (validates inclusion:/
// exclusion:): either a discrete In set or a numeric Range.
type MembershipOptions struct {
	In      []any
	Range   *NumRange
	Message any
}

// NumericalityOptions configures the numericality validator. Each comparison
// bound may be a literal number, a func(Model) any (a Ruby proc), or a Symbol (a
// method name resolved through the Dispatcher seam).
type NumericalityOptions struct {
	OnlyInteger          bool
	GreaterThan          any
	GreaterThanOrEqualTo any
	EqualTo              any
	LessThan             any
	LessThanOrEqualTo    any
	OtherThan            any
	Odd                  bool
	Even                 bool
	Message              any
}

// ConfirmationOptions configures the confirmation validator. CaseSensitive
// defaults to true.
type ConfirmationOptions struct {
	CaseSensitive *bool
	Message       any
}

// AcceptanceOptions configures the acceptance validator. Accept defaults to
// [true, "1"]; acceptance skips nil values (allow_nil defaults true in Rails).
type AcceptanceOptions struct {
	Accept  []any
	Message any
}

// msgFor prefers a validator-specific message over the shared one.
func msgFor(specific, shared any) any {
	if specific != nil {
		return specific
	}
	return shared
}

// msgOpts builds the options map carrying an optional :message override.
func msgOpts(msg any) map[string]any {
	if msg == nil {
		return map[string]any{}
	}
	return map[string]any{"message": msg}
}

// toFloat coerces a numeric value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// containsAny reports whether set contains value under Ruby-ish equality.
func containsAny(set []any, value any) bool {
	for _, e := range set {
		if reflect.DeepEqual(e, value) {
			return true
		}
	}
	return false
}

// ---- presence / absence ----

func validatePresence(e *Errors, attr string, value, msg any) {
	if isBlank(value) {
		e.Add(attr, Symbol("blank"), msgOpts(msg))
	}
}

func validateAbsence(e *Errors, attr string, value, msg any) {
	if isPresent(value) {
		e.Add(attr, Symbol("present"), msgOpts(msg))
	}
}

// ---- acceptance ----

func validateAcceptance(e *Errors, attr string, value any, o *AcceptanceOptions, shared any) {
	accept := o.Accept
	if accept == nil {
		accept = []any{true, "1"}
	}
	if !containsAny(accept, value) {
		e.Add(attr, Symbol("accepted"), msgOpts(msgFor(o.Message, shared)))
	}
}

// ---- confirmation ----

func validateConfirmation(m Model, e *Errors, attr string, value any, o *ConfirmationOptions, shared any) {
	confirmed := m.Get(attr + "_confirmation")
	if confirmed == nil {
		return
	}
	caseSensitive := o.CaseSensitive == nil || *o.CaseSensitive
	var equal bool
	if caseSensitive {
		equal = reflect.DeepEqual(value, confirmed)
	} else {
		equal = strings.EqualFold(rubyString(value), rubyString(confirmed))
	}
	if !equal {
		opts := map[string]any{"attribute": humanAttributeName(attr)}
		if msg := msgFor(o.Message, shared); msg != nil {
			opts["message"] = msg
		}
		e.Add(attr+"_confirmation", Symbol("confirmation"), opts)
	}
}

// ---- length ----

func valueLength(v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case string:
		return utf8.RuneCountInString(t)
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Map:
			return rv.Len()
		default:
			return utf8.RuneCountInString(rubyString(v))
		}
	}
}

func addLength(e *Errors, attr, key string, count int, specific, shared any) {
	opts := map[string]any{"count": count}
	if msg := msgFor(specific, shared); msg != nil {
		opts["message"] = msg
	}
	e.Add(attr, Symbol(key), opts)
}

func validateLength(e *Errors, attr string, value any, o *LengthOptions, shared any) {
	length := valueLength(value)
	if o.Is != nil && length != *o.Is {
		addLength(e, attr, "wrong_length", *o.Is, o.WrongLength, shared)
	}
	min, max := o.Minimum, o.Maximum
	if o.In != nil {
		lo, hi := o.In.Min, o.In.Max
		min, max = &lo, &hi
	}
	if min != nil && length < *min {
		addLength(e, attr, "too_short", *min, o.TooShort, shared)
	}
	if max != nil && length > *max {
		addLength(e, attr, "too_long", *max, o.TooLong, shared)
	}
}

// ---- format ----

func validateFormat(e *Errors, attr string, value any, o *FormatOptions, shared any) {
	s := rubyString(value)
	if o.With != nil && !o.With.MatchString(s) {
		e.Add(attr, Symbol("invalid"), msgOpts(msgFor(o.Message, shared)))
	}
	if o.Without != nil && o.Without.MatchString(s) {
		e.Add(attr, Symbol("invalid"), msgOpts(msgFor(o.Message, shared)))
	}
}

// ---- inclusion / exclusion ----

func membershipInclude(o *MembershipOptions, value any) bool {
	if o.Range != nil {
		return o.Range.cover(value)
	}
	return containsAny(o.In, value)
}

func membershipOpts(value, msg any) map[string]any {
	opts := map[string]any{"value": value}
	if msg != nil {
		opts["message"] = msg
	}
	return opts
}

func validateInclusion(e *Errors, attr string, value any, o *MembershipOptions, shared any) {
	if !membershipInclude(o, value) {
		e.Add(attr, Symbol("inclusion"), membershipOpts(value, msgFor(o.Message, shared)))
	}
}

func validateExclusion(e *Errors, attr string, value any, o *MembershipOptions, shared any) {
	if membershipInclude(o, value) {
		e.Add(attr, Symbol("exclusion"), membershipOpts(value, msgFor(o.Message, shared)))
	}
}

// ---- numericality ----

var reInteger = regexp.MustCompile(`^[+-]?\d+$`)

// parseNumeric mirrors ActiveModel's number parsing: a Go integer is an integer,
// a Go float is a number but not an integer, and a string is parsed as an integer
// when it matches /\A[+-]?\d+\z/ else as a float (Kernel.Float).
func parseNumeric(value any) (f float64, isInt bool, ok bool) {
	switch t := value.(type) {
	case int:
		return float64(t), true, true
	case int64:
		return float64(t), true, true
	case int32:
		return float64(t), true, true
	case float64:
		return t, false, true
	case float32:
		return float64(t), false, true
	case string:
		s := strings.TrimSpace(t)
		if reInteger.MatchString(s) {
			n, _ := strconv.ParseInt(s, 10, 64)
			return float64(n), true, true
		}
		fv, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false, false
		}
		return fv, false, true
	default:
		return 0, false, false
	}
}

// resolveBound resolves a comparison bound that may be a literal, a proc, or a
// method-name Symbol.
func resolveBound(m Model, b any) any {
	switch t := b.(type) {
	case func(Model) any:
		return t(m)
	case Symbol:
		return m.Call(string(t))
	default:
		return b
	}
}

func numOpts(value, msg any) map[string]any {
	opts := map[string]any{"value": value}
	if msg != nil {
		opts["message"] = msg
	}
	return opts
}

func validateNumericality(m Model, e *Errors, attr string, value any, o *NumericalityOptions, shared any) {
	msg := msgFor(o.Message, shared)
	f, isInt, ok := parseNumeric(value)
	if !ok {
		e.Add(attr, Symbol("not_a_number"), numOpts(value, msg))
		return
	}
	if o.OnlyInteger && !isInt {
		e.Add(attr, Symbol("not_an_integer"), numOpts(value, msg))
		return
	}

	checks := []struct {
		bound any
		key   string
		ok    func(a, b float64) bool
	}{
		{o.GreaterThan, "greater_than", func(a, b float64) bool { return a > b }},
		{o.GreaterThanOrEqualTo, "greater_than_or_equal_to", func(a, b float64) bool { return a >= b }},
		{o.EqualTo, "equal_to", func(a, b float64) bool { return a == b }},
		{o.LessThan, "less_than", func(a, b float64) bool { return a < b }},
		{o.LessThanOrEqualTo, "less_than_or_equal_to", func(a, b float64) bool { return a <= b }},
		{o.OtherThan, "other_than", func(a, b float64) bool { return a != b }},
	}
	for _, c := range checks {
		if c.bound == nil {
			continue
		}
		bv := resolveBound(m, c.bound)
		bf, _ := toFloat(bv)
		if !c.ok(f, bf) {
			opts := numOpts(value, msg)
			opts["count"] = bv
			e.Add(attr, Symbol(c.key), opts)
		}
	}

	// odd / even operate on to_i, and come after the comparisons in Rails' CHECKS.
	i := int64(f)
	if o.Odd && i%2 == 0 {
		e.Add(attr, Symbol("odd"), numOpts(value, msg))
	}
	if o.Even && i%2 != 0 {
		e.Add(attr, Symbol("even"), numOpts(value, msg))
	}
}
