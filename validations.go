// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

// Condition is a single if:/unless: predicate. Method names a model method
// resolved through the Dispatcher seam (a Ruby symbol condition); otherwise Proc
// is called (a Ruby proc/lambda). Either way the result is judged by Ruby
// truthiness.
type Condition struct {
	Method string
	Proc   func(Model) any
}

func (c Condition) eval(m Model) bool {
	if c.Method != "" {
		return truthy(m.Call(c.Method))
	}
	return truthy(c.Proc(m))
}

// MethodCond builds a symbol condition naming a model method.
func MethodCond(method string) Condition { return Condition{Method: method} }

// ProcCond builds a proc condition from a Go predicate.
func ProcCond(fn func(Model) any) Condition { return Condition{Proc: fn} }

// Conditions is the shared control block for a validator: if:/unless:/on: plus
// allow_nil:/allow_blank:.
type Conditions struct {
	If         []Condition
	Unless     []Condition
	On         []string
	AllowNil   bool
	AllowBlank bool
}

// pass reports whether the if/unless/on gates admit running in the given context.
func (c Conditions) pass(m Model, context string) bool {
	for _, cond := range c.If {
		if !cond.eval(m) {
			return false
		}
	}
	for _, cond := range c.Unless {
		if cond.eval(m) {
			return false
		}
	}
	if len(c.On) > 0 {
		found := false
		for _, o := range c.On {
			if o == context {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Options is the bundle passed to Validates: which standard validators to apply
// to the attribute(s), plus the shared control keys. It mirrors the Ruby hash
// `validates :attr, presence: true, length: {...}, if: ..., allow_blank: true`.
type Options struct {
	Presence     bool
	Absence      bool
	Length       *LengthOptions
	Format       *FormatOptions
	Inclusion    *MembershipOptions
	Exclusion    *MembershipOptions
	Numericality *NumericalityOptions
	Confirmation *ConfirmationOptions
	Acceptance   *AcceptanceOptions

	Message    any
	AllowNil   bool
	AllowBlank bool
	If         []Condition
	Unless     []Condition
	On         []string
}

// Validator is ActiveModel::Validator: a whole-record custom validator.
type Validator interface {
	Validate(m Model, e *Errors)
}

// EachValidator is ActiveModel::EachValidator: a custom validator invoked once
// per attribute with its value (after allow_nil/allow_blank filtering).
type EachValidator interface {
	ValidateEach(m Model, e *Errors, attribute string, value any)
}

// registration is one entry in the validation chain: either a whole-record
// callback or a per-attribute one, gated by its conditions.
type registration struct {
	conds      Conditions
	attributes []string
	each       func(m Model, e *Errors, attribute string, value any)
	whole      func(m Model, e *Errors)
}

func (r registration) run(m Model, e *Errors, context string) {
	if !r.conds.pass(m, context) {
		return
	}
	if r.whole != nil {
		r.whole(m, e)
		return
	}
	for _, attr := range r.attributes {
		value := m.Get(attr)
		if r.conds.AllowNil && value == nil {
			continue
		}
		if r.conds.AllowBlank && isBlank(value) {
			continue
		}
		r.each(m, e, attr, value)
	}
}

// Validations is the ActiveModel::Validations engine for one model class: it
// holds the registered validators and the model's Name.
type Validations struct {
	name Name
	regs []registration
}

// New builds a Validations bound to a model Name (used for %{model} and the
// Errors it produces).
func New(name Name) *Validations { return &Validations{name: name} }

// ModelName returns the model's Name (ActiveModel::Naming#model_name).
func (v *Validations) ModelName() Name { return v.name }

// Validates is ActiveModel::Validations#validates: register the configured
// standard validators for the given attribute(s).
func (v *Validations) Validates(attrs []string, opts Options) {
	base := Conditions{
		If: opts.If, Unless: opts.Unless, On: opts.On,
		AllowNil: opts.AllowNil, AllowBlank: opts.AllowBlank,
	}
	add := func(c Conditions, each func(m Model, e *Errors, attribute string, value any)) {
		v.regs = append(v.regs, registration{conds: c, attributes: attrs, each: each})
	}

	if opts.Presence {
		msg := opts.Message
		add(base, func(m Model, e *Errors, attr string, value any) {
			validatePresence(e, attr, value, msg)
		})
	}
	if opts.Absence {
		msg := opts.Message
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateAbsence(e, attr, value, msg)
		})
	}
	if o := opts.Length; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateLength(e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Format; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateFormat(e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Inclusion; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateInclusion(e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Exclusion; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateExclusion(e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Numericality; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateNumericality(m, e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Confirmation; o != nil {
		add(base, func(m Model, e *Errors, attr string, value any) {
			validateConfirmation(m, e, attr, value, o, opts.Message)
		})
	}
	if o := opts.Acceptance; o != nil {
		// acceptance defaults allow_nil: true.
		c := base
		c.AllowNil = true
		add(c, func(m Model, e *Errors, attr string, value any) {
			validateAcceptance(e, attr, value, o, opts.Message)
		})
	}
}

// Validate is ActiveModel::Validations#validate with a block: register a
// whole-record custom validation.
func (v *Validations) Validate(c Conditions, block func(m Model, e *Errors)) {
	v.regs = append(v.regs, registration{conds: c, whole: block})
}

// ValidatesEach is ActiveModel::Validations#validates_each: run block once per
// attribute with its value, after allow_nil/allow_blank filtering.
func (v *Validations) ValidatesEach(attrs []string, c Conditions, block func(m Model, e *Errors, attribute string, value any)) {
	v.regs = append(v.regs, registration{conds: c, attributes: attrs, each: block})
}

// ValidatesWith registers a whole-record custom Validator
// (ActiveModel::Validations#validates_with).
func (v *Validations) ValidatesWith(validator Validator, c Conditions) {
	v.regs = append(v.regs, registration{conds: c, whole: validator.Validate})
}

// ValidatesEachWith registers a custom EachValidator over the given attributes.
func (v *Validations) ValidatesEachWith(attrs []string, validator EachValidator, c Conditions) {
	v.regs = append(v.regs, registration{conds: c, attributes: attrs, each: validator.ValidateEach})
}

// ValidContext is ActiveModel::Validations#valid? in the given context: run every
// validator against the model, collecting errors, and report whether none were
// added.
func (v *Validations) ValidContext(m Model, context string) (bool, *Errors) {
	e := NewErrors(m, v.name)
	for _, r := range v.regs {
		r.run(m, e, context)
	}
	return e.Empty(), e
}

// Valid runs validation in the default (empty) context.
func (v *Validations) Valid(m Model) (bool, *Errors) { return v.ValidContext(m, "") }

// InvalidContext is ActiveModel::Validations#invalid? in the given context.
func (v *Validations) InvalidContext(m Model, context string) (bool, *Errors) {
	ok, e := v.ValidContext(m, context)
	return !ok, e
}

// Invalid runs invalidation in the default context.
func (v *Validations) Invalid(m Model) (bool, *Errors) {
	ok, e := v.Valid(m)
	return !ok, e
}
