// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"reflect"
	"regexp"
)

// defaultMessages is ActiveModel's en.yml errors.messages table (the count-free
// entries). The count-sensitive length entries live in pluralMessages.
var defaultMessages = map[string]string{
	"inclusion":                "is not included in the list",
	"exclusion":                "is reserved",
	"invalid":                  "is invalid",
	"confirmation":             "doesn't match %{attribute}",
	"accepted":                 "must be accepted",
	"empty":                    "can't be empty",
	"blank":                    "can't be blank",
	"present":                  "must be blank",
	"not_a_number":             "is not a number",
	"not_an_integer":           "must be an integer",
	"greater_than":             "must be greater than %{count}",
	"greater_than_or_equal_to": "must be greater than or equal to %{count}",
	"equal_to":                 "must be equal to %{count}",
	"less_than":                "must be less than %{count}",
	"less_than_or_equal_to":    "must be less than or equal to %{count}",
	"other_than":               "must be other than %{count}",
	"odd":                      "must be odd",
	"even":                     "must be even",
	"model_invalid":            "Validation failed: %{errors}",
}

// pluralMessages holds the [one, other] forms of the count-sensitive length
// messages, matching ActiveModel's I18n pluralization (count == 1 uses the
// singular "character").
var pluralMessages = map[string][2]string{
	"too_long":     {"is too long (maximum is 1 character)", "is too long (maximum is %{count} characters)"},
	"too_short":    {"is too short (minimum is 1 character)", "is too short (minimum is %{count} characters)"},
	"wrong_length": {"is the wrong length (should be 1 character)", "is the wrong length (should be %{count} characters)"},
}

// callbackOptions are the validator control keys that never take part in a
// message, a detail, or a strict match (ActiveModel::Error::CALLBACKS_OPTIONS).
var callbackOptions = map[string]bool{
	"if": true, "unless": true, "on": true,
	"allow_nil": true, "allow_blank": true, "strict": true,
}

func isCallbackOption(k string) bool { return callbackOptions[k] }

var reInterpolate = regexp.MustCompile(`%\{(\w+)\}`)

// interpolate substitutes each %{key} in tmpl with rubyString(vars[key]); an
// unknown key is left untouched.
func interpolate(tmpl string, vars map[string]any) string {
	return reInterpolate.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := m[2 : len(m)-1]
		if v, ok := vars[key]; ok {
			return rubyString(v)
		}
		return m
	})
}

// toInt coerces a numeric option to an int for pluralization selection.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// lookupTemplate resolves the message template for a symbol key, honouring the
// count-based pluralization of the length messages and Rails' "translation
// missing" fallback for an unknown key.
func lookupTemplate(key string, opts map[string]any) string {
	if pm, ok := pluralMessages[key]; ok {
		if toInt(opts["count"]) == 1 {
			return pm[0]
		}
		return pm[1]
	}
	if m, ok := defaultMessages[key]; ok {
		return m
	}
	return "translation missing: en.errors.messages." + key
}

// Error is ActiveModel::Error: a single validation error bound to an attribute,
// with a (Symbol or literal-String) type and its options.
type Error struct {
	base      Model
	name      Name
	attribute string
	rawType   any
	options   map[string]any
}

// Attribute returns the attribute the error is on.
func (e *Error) Attribute() string { return e.attribute }

// Type returns the raw error type (a Symbol or a literal String), defaulting to
// Symbol("invalid") when none was given.
func (e *Error) Type() any {
	if e.rawType == nil {
		return Symbol("invalid")
	}
	return e.rawType
}

// Options returns the error's options.
func (e *Error) Options() map[string]any { return e.options }

// attrValue is the model's value for the attribute, used for the %{value}
// interpolation, following ActiveModel's respond_to? guard.
func (e *Error) attrValue() any {
	if e.attribute != "base" && e.base != nil && e.base.RespondTo(e.attribute) {
		return e.base.Get(e.attribute)
	}
	return nil
}

// vars builds the interpolation variables for the given key: the special model /
// attribute / value bindings, overlaid by the (non-callback) options so that a
// per-error override such as confirmation's attribute: wins.
func (e *Error) vars() map[string]any {
	v := map[string]any{
		"model":     e.name.Human,
		"attribute": humanAttributeName(e.attribute),
		"value":     e.attrValue(),
	}
	for k, val := range e.options {
		if k == "message" || isCallbackOption(k) {
			continue
		}
		v[k] = val
	}
	return v
}

// Message is ActiveModel::Error#message: a literal String type verbatim, or the
// generated, interpolated message for a Symbol type.
func (e *Error) Message() string {
	switch rt := e.rawType.(type) {
	case string:
		return rt
	case Symbol:
		return e.generateMessage(string(rt))
	default:
		return rubyString(rt)
	}
}

// generateMessage is ActiveModel::Error.generate_message: a :message Symbol
// option overrides the key, a String option is used as an interpolated template,
// otherwise the default table entry for key is interpolated.
func (e *Error) generateMessage(key string) string {
	if m, ok := e.options["message"]; ok {
		switch mv := m.(type) {
		case Symbol:
			key = string(mv)
		case string:
			return interpolate(mv, e.vars())
		case func(Model) string:
			return mv(e.base)
		}
	}
	return interpolate(lookupTemplate(key, e.options), e.vars())
}

// FullMessage is ActiveModel::Error#full_message: "%{attribute} %{message}",
// except a :base error is its message alone.
func (e *Error) FullMessage() string {
	if e.attribute == "base" {
		return e.Message()
	}
	return humanAttributeName(e.attribute) + " " + e.Message()
}

// Details is ActiveModel::Error#details: {error: type, ...options} minus the
// callback and message options.
func (e *Error) Details() map[string]any {
	d := map[string]any{"error": e.Type()}
	for k, v := range e.options {
		if k == "message" || isCallbackOption(k) {
			continue
		}
		d[k] = v
	}
	return d
}

// Match is ActiveModel::Error#match?: same attribute, optionally the same type,
// and every supplied option equal.
func (e *Error) Match(attribute string, typ any, opts map[string]any) bool {
	if e.attribute != attribute {
		return false
	}
	if typ != nil && !typeEqual(e.rawType, typ) {
		return false
	}
	for k, v := range opts {
		if !reflect.DeepEqual(e.options[k], v) {
			return false
		}
	}
	return true
}

// strictMatch is ActiveModel::Error#strict_match?: Match plus opts being exactly
// the error's non-callback, non-message options.
func (e *Error) strictMatch(attribute string, typ any, opts map[string]any) bool {
	if !e.Match(attribute, typ, opts) {
		return false
	}
	filtered := map[string]any{}
	for k, v := range e.options {
		if k == "message" || isCallbackOption(k) {
			continue
		}
		filtered[k] = v
	}
	if opts == nil {
		opts = map[string]any{}
	}
	return reflect.DeepEqual(filtered, opts)
}

// typeEqual compares two error types where each may be a Symbol or a String.
func typeEqual(a, b any) bool {
	return rubyString(a) == rubyString(b) && symbolness(a) == symbolness(b)
}

func symbolness(v any) bool {
	_, ok := v.(Symbol)
	return ok
}

// Errors is ActiveModel::Errors: the ordered collection of an object's errors.
type Errors struct {
	base    Model
	name    Name
	entries []*Error
}

// NewErrors builds an Errors bound to a model and its Name (used for the %{model}
// interpolation and full_message attribute humanization).
func NewErrors(base Model, name Name) *Errors {
	return &Errors{base: base, name: name}
}

// normalizeType resolves a proc type to its result and defaults a nil type to
// Symbol("invalid"), mirroring Errors#normalize_arguments.
func normalizeType(base Model, typ any) any {
	if p, ok := typ.(func(Model) any); ok {
		typ = p(base)
	}
	if typ == nil {
		return Symbol("invalid")
	}
	return typ
}

// Add is ActiveModel::Errors#add: append an error on attribute with the given
// type and options and return it. A nil type defaults to Symbol("invalid").
func (es *Errors) Add(attribute string, typ any, opts map[string]any) *Error {
	e := &Error{
		base:      es.base,
		name:      es.name,
		attribute: attribute,
		rawType:   normalizeType(es.base, typ),
		options:   opts,
	}
	if e.options == nil {
		e.options = map[string]any{}
	}
	es.entries = append(es.entries, e)
	return e
}

// Clear empties the collection (Errors#clear).
func (es *Errors) Clear() { es.entries = nil }

// Empty reports Errors#empty? / #blank?.
func (es *Errors) Empty() bool { return len(es.entries) == 0 }

// Any reports Errors#any?.
func (es *Errors) Any() bool { return len(es.entries) > 0 }

// Size is Errors#size / #count.
func (es *Errors) Size() int { return len(es.entries) }

// Entries returns the underlying ordered errors (Errors#errors).
func (es *Errors) Entries() []*Error { return es.entries }

// Each iterates the errors in order (Errors#each).
func (es *Errors) Each(fn func(*Error)) {
	for _, e := range es.entries {
		fn(e)
	}
}

// Include reports whether any error is on attribute (Errors#include? / #has_key?).
func (es *Errors) Include(attribute string) bool {
	for _, e := range es.entries {
		if e.attribute == attribute {
			return true
		}
	}
	return false
}

// Where is Errors#where: the errors matching attribute and, when given, type and
// options.
func (es *Errors) Where(attribute string, typ any, opts map[string]any) []*Error {
	var out []*Error
	for _, e := range es.entries {
		if e.Match(attribute, typ, opts) {
			out = append(out, e)
		}
	}
	return out
}

// Added is ActiveModel::Errors#added?: for a Symbol type, whether an error
// strictly matches attribute/type/options; for a literal String type, whether
// that message is present on the attribute.
func (es *Errors) Added(attribute string, typ any, opts map[string]any) bool {
	if _, isStr := typ.(string); isStr {
		want := rubyString(typ)
		for _, m := range es.MessagesFor(attribute) {
			if m == want {
				return true
			}
		}
		return false
	}
	typ = normalizeType(es.base, typ)
	for _, e := range es.entries {
		if e.strictMatch(attribute, typ, opts) {
			return true
		}
	}
	return false
}

// OfKind is ActiveModel::Errors#of_kind?: like Added but ignoring options for a
// Symbol type.
func (es *Errors) OfKind(attribute string, typ any) bool {
	if _, isStr := typ.(string); isStr {
		want := rubyString(typ)
		for _, m := range es.MessagesFor(attribute) {
			if m == want {
				return true
			}
		}
		return false
	}
	typ = normalizeType(es.base, typ)
	return len(es.Where(attribute, typ, nil)) > 0
}

// MessagesFor is Errors#messages_for(attribute): the message strings on the
// attribute, in order.
func (es *Errors) MessagesFor(attribute string) []string {
	var out []string
	for _, e := range es.entries {
		if e.attribute == attribute {
			out = append(out, e.Message())
		}
	}
	return out
}

// Get is ActiveModel::Errors#[]: the message strings on the attribute (an alias
// for MessagesFor).
func (es *Errors) Get(attribute string) []string { return es.MessagesFor(attribute) }

// FullMessagesFor is Errors#full_messages_for(attribute).
func (es *Errors) FullMessagesFor(attribute string) []string {
	var out []string
	for _, e := range es.entries {
		if e.attribute == attribute {
			out = append(out, e.FullMessage())
		}
	}
	return out
}

// FullMessages is Errors#full_messages: every error's full message, in order.
func (es *Errors) FullMessages() []string {
	out := make([]string, 0, len(es.entries))
	for _, e := range es.entries {
		out = append(out, e.FullMessage())
	}
	return out
}

// Messages is Errors#messages: attribute => its message strings.
func (es *Errors) Messages() map[string][]string {
	out := map[string][]string{}
	for _, e := range es.entries {
		out[e.attribute] = append(out[e.attribute], e.Message())
	}
	return out
}

// Details is Errors#details: attribute => its detail hashes.
func (es *Errors) Details() map[string][]map[string]any {
	out := map[string][]map[string]any{}
	for _, e := range es.entries {
		out[e.attribute] = append(out[e.attribute], e.Details())
	}
	return out
}

// Attribute names present in the collection, in first-seen order
// (Errors#attribute_names).
func (es *Errors) AttributeNames() []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range es.entries {
		if !seen[e.attribute] {
			seen[e.attribute] = true
			out = append(out, e.attribute)
		}
	}
	return out
}
