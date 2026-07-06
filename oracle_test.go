// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

var reAt = regexp.MustCompile(`@`)

// rubyAM locates a ruby that can `require "active_model"`, skipping the oracle
// otherwise (the qemu cross-arch lanes and any host without the gem). The
// deterministic suite alone holds coverage at 100%, so skipping here never
// weakens the gate.
func rubyAM(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping ActiveModel oracle")
	}
	if err := exec.Command(path, "-e", "require 'active_model'").Run(); err != nil {
		t.Skip("active_model gem not installed; skipping ActiveModel oracle")
	}
	return path
}

// rubyEmit runs a ruby script (binmoded, per the go-ruby-erb Windows lesson) and
// returns id -> value parsed from its "id\tvalue" lines.
func rubyEmit(t *testing.T, bin, script string) map[string]string {
	t.Helper()
	full := "$stdin.binmode\n$stdout.binmode\nrequire 'active_model'\n" +
		"def emit(id, v); puts \"#{id}\\t#{v}\"; end\n" + script
	out, err := exec.Command(bin, "-e", full).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\noutput:\n%s", err, out)
	}
	res := map[string]string{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			res[parts[0]] = parts[1]
		}
	}
	return res
}

func TestOracleNaming(t *testing.T) {
	bin := rubyAM(t)
	script := `
class Person; extend ActiveModel::Naming; end
module Admin; class User; extend ActiveModel::Naming; end; end
class LineItem; extend ActiveModel::Naming; end
def name_fields(k)
  n = k.model_name
  [n.singular, n.plural, n.element, n.human, n.collection, n.param_key, n.route_key, n.singular_route_key, n.i18n_key].join("|")
end
emit("Person", name_fields(Person))
emit("Admin::User", name_fields(Admin::User))
emit("LineItem", name_fields(LineItem))
`
	want := rubyEmit(t, bin, script)
	for _, cls := range []string{"Person", "Admin::User", "LineItem"} {
		n := NewName(cls)
		got := strings.Join([]string{
			n.Singular, n.Plural, n.Element, n.Human, n.Collection,
			n.ParamKey, n.RouteKey, n.SingularRouteKey, n.I18nKey,
		}, "|")
		if got != want[cls] {
			t.Errorf("naming %s\n go  %q\n ruby %q", cls, got, want[cls])
		}
	}
}

// oracleCase pairs a Ruby model/validation with the equivalent Go build, so the
// two produce the same full_messages.
type oracleCase struct {
	id   string
	ruby string
	run  func() []string
}

func fullMessagesOf(v *Validations, m Model) []string {
	_, e := v.Valid(m)
	return e.FullMessages()
}

func TestOracleValidations(t *testing.T) {
	bin := rubyAM(t)

	cases := []oracleCase{
		{
			id:   "presence",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :name;validates :name,presence:true};m=c.new(name:"");m.valid?;emit("presence",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"name"}, Options{Presence: true})
				return fullMessagesOf(v, newModel().with("name", ""))
			},
		},
		{
			id:   "length_min",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :name;validates :name,length:{minimum:2}};m=c.new(name:"a");m.valid?;emit("length_min",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"name"}, Options{Length: &LengthOptions{Minimum: intp(2)}})
				return fullMessagesOf(v, newModel().with("name", "a"))
			},
		},
		{
			id:   "length_is",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :code;validates :code,length:{is:1}};m=c.new(code:"ab");m.valid?;emit("length_is",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"code"}, Options{Length: &LengthOptions{Is: intp(1)}})
				return fullMessagesOf(v, newModel().with("code", "ab"))
			},
		},
		{
			id:   "format",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :email;validates :email,format:{with:/@/}};m=c.new(email:"x");m.valid?;emit("format",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"email"}, Options{Format: &FormatOptions{With: reAt}})
				return fullMessagesOf(v, newModel().with("email", "x"))
			},
		},
		{
			id:   "inclusion",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :size;validates :size,inclusion:{in:%w[s m l]}};m=c.new(size:"xl");m.valid?;emit("inclusion",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"size"}, Options{Inclusion: &MembershipOptions{In: []any{"s", "m", "l"}}})
				return fullMessagesOf(v, newModel().with("size", "xl"))
			},
		},
		{
			id:   "exclusion",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :name;validates :name,exclusion:{in:%w[admin]}};m=c.new(name:"admin");m.valid?;emit("exclusion",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"name"}, Options{Exclusion: &MembershipOptions{In: []any{"admin"}}})
				return fullMessagesOf(v, newModel().with("name", "admin"))
			},
		},
		{
			id:   "num_nan",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :age;validates :age,numericality:true};m=c.new(age:"abc");m.valid?;emit("num_nan",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"age"}, Options{Numericality: &NumericalityOptions{}})
				return fullMessagesOf(v, newModel().with("age", "abc"))
			},
		},
		{
			id:   "num_int",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :age;validates :age,numericality:{only_integer:true}};m=c.new(age:"1.5");m.valid?;emit("num_int",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"age"}, Options{Numericality: &NumericalityOptions{OnlyInteger: true}})
				return fullMessagesOf(v, newModel().with("age", "1.5"))
			},
		},
		{
			id:   "num_gt",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :age;validates :age,numericality:{greater_than:18}};m=c.new(age:10);m.valid?;emit("num_gt",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"age"}, Options{Numericality: &NumericalityOptions{GreaterThan: 18}})
				return fullMessagesOf(v, newModel().with("age", 10))
			},
		},
		{
			id:   "num_multi",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :n;validates :n,numericality:{equal_to:3,odd:true}};m=c.new(n:4);m.valid?;emit("num_multi",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"n"}, Options{Numericality: &NumericalityOptions{EqualTo: 3, Odd: true}})
				return fullMessagesOf(v, newModel().with("n", 4))
			},
		},
		{
			id:   "confirmation",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :password,:password_confirmation;validates :password,confirmation:true};m=c.new(password:"a",password_confirmation:"b");m.valid?;emit("confirmation",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"password"}, Options{Confirmation: &ConfirmationOptions{}})
				return fullMessagesOf(v, newModel().with("password", "a").with("password_confirmation", "b"))
			},
		},
		{
			id:   "acceptance",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :tos;validates :tos,acceptance:true};m=c.new(tos:"0");m.valid?;emit("acceptance",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"tos"}, Options{Acceptance: &AcceptanceOptions{}})
				return fullMessagesOf(v, newModel().with("tos", "0"))
			},
		},
		{
			id:   "absence",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :name;validates :name,absence:true};m=c.new(name:"x");m.valid?;emit("absence",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"name"}, Options{Absence: true})
				return fullMessagesOf(v, newModel().with("name", "x"))
			},
		},
		{
			id:   "custom_msg",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :name;validates :name,presence:{message:"is required"}};m=c.new(name:"");m.valid?;emit("custom_msg",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"name"}, Options{Presence: true, Message: "is required"})
				return fullMessagesOf(v, newModel().with("name", ""))
			},
		},
		{
			id:   "custom_interp",
			ruby: `c=Class.new{include ActiveModel::Model;def self.name;"Person";end;attr_accessor :n;validates :n,length:{maximum:3,message:"too long: %{count}"}};m=c.new(n:"abcd");m.valid?;emit("custom_interp",m.errors.full_messages.join(" | "))`,
			run: func() []string {
				v := New(NewName("Person"))
				v.Validates([]string{"n"}, Options{Length: &LengthOptions{Maximum: intp(3)}, Message: "too long: %{count}"})
				return fullMessagesOf(v, newModel().with("n", "abcd"))
			},
		},
	}

	var script strings.Builder
	for _, c := range cases {
		script.WriteString(c.ruby)
		script.WriteString("\n")
	}
	want := rubyEmit(t, bin, script.String())

	for _, c := range cases {
		got := strings.Join(c.run(), " | ")
		if got != want[c.id] {
			t.Errorf("case %s\n go   %q\n ruby %q", c.id, got, want[c.id])
		}
	}
}
