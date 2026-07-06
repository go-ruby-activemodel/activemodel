// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"reflect"
	"regexp"
	"testing"
)

func personValidations() *Validations { return New(NewName("Person")) }

func TestConditionEval(t *testing.T) {
	m := newModel()
	m.methods["admin?"] = true
	if !MethodCond("admin?").eval(m) {
		t.Error("method cond true")
	}
	if !ProcCond(func(Model) any { return 1 }).eval(m) {
		t.Error("proc cond truthy")
	}
	if ProcCond(func(Model) any { return nil }).eval(m) {
		t.Error("proc cond falsy")
	}
}

func TestConditionsPass(t *testing.T) {
	m := newModel()
	m.methods["yes"] = true
	m.methods["no"] = false
	// if fails
	if (Conditions{If: []Condition{MethodCond("no")}}).pass(m, "") {
		t.Error("if false should not pass")
	}
	// unless true fails
	if (Conditions{Unless: []Condition{MethodCond("yes")}}).pass(m, "") {
		t.Error("unless true should not pass")
	}
	// on mismatch
	if (Conditions{On: []string{"create"}}).pass(m, "") {
		t.Error("on mismatch should not pass")
	}
	// on match
	if !(Conditions{On: []string{"create"}}).pass(m, "create") {
		t.Error("on match should pass")
	}
	// all pass
	if !(Conditions{If: []Condition{MethodCond("yes")}, Unless: []Condition{MethodCond("no")}}).pass(m, "") {
		t.Error("all conditions should pass")
	}
}

func TestValidatesPresenceEngine(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"name"}, Options{Presence: true})
	ok, e := v.Valid(newModel().with("name", ""))
	if ok || e.FullMessages()[0] != "Name can't be blank" {
		t.Errorf("presence engine = %v %v", ok, e.FullMessages())
	}
	ok2, _ := v.Valid(newModel().with("name", "Bob"))
	if !ok2 {
		t.Error("valid model should pass")
	}
	if v.ModelName().Name != "Person" {
		t.Error("model name")
	}
}

func TestValidatesAllKinds(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"name"}, Options{Presence: true})
	v.Validates([]string{"secret"}, Options{Absence: true})
	v.Validates([]string{"bio"}, Options{Length: &LengthOptions{Maximum: intp(3)}})
	v.Validates([]string{"email"}, Options{Format: &FormatOptions{With: regexp.MustCompile(`@`)}})
	v.Validates([]string{"size"}, Options{Inclusion: &MembershipOptions{In: []any{"s"}}})
	v.Validates([]string{"user"}, Options{Exclusion: &MembershipOptions{In: []any{"root"}}})
	v.Validates([]string{"age"}, Options{Numericality: &NumericalityOptions{OnlyInteger: true}})
	v.Validates([]string{"password"}, Options{Confirmation: &ConfirmationOptions{}})
	v.Validates([]string{"tos"}, Options{Acceptance: &AcceptanceOptions{}})

	m := newModel().
		with("name", "").
		with("secret", "leak").
		with("bio", "toolong").
		with("email", "nope").
		with("size", "xl").
		with("user", "root").
		with("age", "1.5").
		with("password", "a").with("password_confirmation", "b").
		with("tos", "0")
	ok, e := v.Valid(m)
	if ok {
		t.Fatal("model should be invalid")
	}
	got := e.FullMessages()
	want := []string{
		"Name can't be blank",
		"Secret must be blank",
		"Bio is too long (maximum is 3 characters)",
		"Email is invalid",
		"Size is not included in the list",
		"User is reserved",
		"Age must be an integer",
		"Password confirmation doesn't match Password",
		"Tos must be accepted",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("all-kinds messages\n got %#v\nwant %#v", got, want)
	}
}

func TestAcceptanceAllowsNil(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"tos"}, Options{Acceptance: &AcceptanceOptions{}})
	// nil is skipped (acceptance defaults allow_nil: true).
	if ok, _ := v.Valid(newModel()); !ok {
		t.Error("acceptance should allow nil")
	}
}

func TestAllowNilAndBlank(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"age"}, Options{Numericality: &NumericalityOptions{}, AllowNil: true})
	if ok, _ := v.Valid(newModel()); !ok {
		t.Error("allow_nil should skip nil")
	}
	v2 := personValidations()
	v2.Validates([]string{"bio"}, Options{Length: &LengthOptions{Minimum: intp(5)}, AllowBlank: true})
	if ok, _ := v2.Valid(newModel().with("bio", "")); !ok {
		t.Error("allow_blank should skip blank")
	}
}

func TestConditionalValidation(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"name"}, Options{
		Presence: true,
		If:       []Condition{ProcCond(func(m Model) any { return m.Get("active") })},
	})
	// condition false -> skipped
	if ok, _ := v.Valid(newModel().with("name", "").with("active", false)); !ok {
		t.Error("if false should skip validation")
	}
	// condition true -> runs
	if ok, _ := v.Valid(newModel().with("name", "").with("active", true)); ok {
		t.Error("if true should run validation")
	}
}

func TestOnContext(t *testing.T) {
	v := personValidations()
	v.Validates([]string{"name"}, Options{Presence: true, On: []string{"create"}})
	if ok, _ := v.Valid(newModel().with("name", "")); !ok {
		t.Error("default context should skip on:create validator")
	}
	if ok, _ := v.ValidContext(newModel().with("name", ""), "create"); ok {
		t.Error("create context should run validator")
	}
	if bad, _ := v.InvalidContext(newModel().with("name", ""), "create"); !bad {
		t.Error("InvalidContext should report invalid")
	}
	if bad, _ := v.Invalid(newModel().with("name", "Bob")); bad {
		t.Error("Invalid should be false for valid model")
	}
}

func TestCustomValidate(t *testing.T) {
	v := personValidations()
	v.Validate(Conditions{}, func(m Model, e *Errors) {
		if m.Get("name") == "forbidden" {
			e.Add("name", "is forbidden", nil)
		}
	})
	if ok, e := v.Valid(newModel().with("name", "forbidden")); ok || e.FullMessages()[0] != "Name is forbidden" {
		t.Errorf("custom validate = %v", e.FullMessages())
	}
	if ok, _ := v.Valid(newModel().with("name", "ok")); !ok {
		t.Error("custom validate should pass")
	}
}

func TestValidatesEach(t *testing.T) {
	v := personValidations()
	v.ValidatesEach([]string{"a", "b"}, Conditions{}, func(m Model, e *Errors, attr string, value any) {
		if value == "bad" {
			e.Add(attr, Symbol("invalid"), nil)
		}
	})
	ok, e := v.Valid(newModel().with("a", "bad").with("b", "good"))
	if ok || len(e.FullMessages()) != 1 || e.FullMessages()[0] != "A is invalid" {
		t.Errorf("validates_each = %v", e.FullMessages())
	}
}

// customValidator implements Validator and EachValidator for the seam tests.
type customValidator struct{}

func (customValidator) Validate(m Model, e *Errors) {
	if m.Get("flag") == true {
		e.Add("base", "whole-record failure", nil)
	}
}

type evenEach struct{}

func (evenEach) ValidateEach(m Model, e *Errors, attr string, value any) {
	if n, ok := value.(int); ok && n%2 != 0 {
		e.Add(attr, Symbol("even"), nil)
	}
}

func TestValidatesWith(t *testing.T) {
	v := personValidations()
	v.ValidatesWith(customValidator{}, Conditions{})
	if ok, e := v.Valid(newModel().with("flag", true)); ok || e.FullMessages()[0] != "whole-record failure" {
		t.Errorf("validates_with = %v", e.FullMessages())
	}
	if ok, _ := v.Valid(newModel().with("flag", false)); !ok {
		t.Error("validates_with should pass")
	}
}

func TestValidatesEachWith(t *testing.T) {
	v := personValidations()
	v.ValidatesEachWith([]string{"n"}, evenEach{}, Conditions{})
	if ok, e := v.Valid(newModel().with("n", 3)); ok || e.FullMessages()[0] != "N must be even" {
		t.Errorf("validates_each_with = %v", e.FullMessages())
	}
}
