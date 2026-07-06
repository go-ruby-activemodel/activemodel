// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"fmt"

	"github.com/go-ruby-activesupport/activesupport/coreext"
)

// Symbol is a Ruby Symbol surfaced to Go. ActiveModel distinguishes a Symbol
// error type (a key looked up in the message table and interpolated) from a
// String error type (a literal message returned verbatim). Passing a Go string
// to Errors.Add / a :message option therefore means a *literal* Ruby String,
// while passing a Symbol("blank") means the Ruby Symbol :blank — exactly the
// MRI distinction in ActiveModel::Error#message.
type Symbol string

// Attr is the attribute-access seam: read and write a model attribute by name.
// It mirrors the Ruby attr reader/writer pair a validated object exposes, and in
// particular ActiveModel's read_attribute_for_validation. A host such as
// go-embedded-ruby plugs its own object here; the tests plug a map-backed fake.
type Attr interface {
	Get(name string) any
	Set(name string, val any)
}

// Dispatcher is the method-call seam for behaviour that is not plain attribute
// access: evaluating a symbol condition (an if:/unless: given as a method name),
// or any host method a validation refers to by name. Call invokes the method and
// returns its Ruby result; RespondTo reports whether the model responds to it
// (used exactly where ActiveModel guards on respond_to?).
type Dispatcher interface {
	Call(method string) any
	RespondTo(method string) bool
}

// Model is the object under validation: the two seams combined.
type Model interface {
	Attr
	Dispatcher
}

// truthy applies Ruby truthiness: only nil and false are falsy.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// isBlank reports Object#blank? for a model value (nil, false, blank string,
// empty array/hash), delegating to ActiveSupport's faithful implementation.
func isBlank(v any) bool { return coreext.Blank(v) }

// isPresent is Object#present?, the inverse of isBlank.
func isPresent(v any) bool { return !coreext.Blank(v) }

// rubyString renders a value the way Ruby's to_s would for the values the
// validators handle: nil is the empty string, a Symbol is its name, a string is
// itself, and everything else uses Go's default formatting (which matches Ruby's
// to_s for integers, floats and booleans).
func rubyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case Symbol:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
