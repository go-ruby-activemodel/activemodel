// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import "testing"

func TestNewName(t *testing.T) {
	n := NewName("Person")
	want := Name{
		Name: "Person", Singular: "person", Plural: "people", Element: "person",
		Human: "Person", Collection: "people", ParamKey: "person",
		RouteKey: "people", SingularRouteKey: "person", I18nKey: "person",
	}
	if n != want {
		t.Errorf("NewName(Person)\n got %+v\nwant %+v", n, want)
	}
}

func TestNewNamespacedName(t *testing.T) {
	// Isolated-engine form: param/route keys drop the namespace (verified
	// against ActiveModel::Name.new(Admin::User, Admin)).
	n := NewNamespacedName("Admin::User", "Admin")
	want := Name{
		Name: "Admin::User", Singular: "admin_user", Plural: "admin_users",
		Element: "user", Human: "User", Collection: "admin/users",
		ParamKey: "user", RouteKey: "users",
		SingularRouteKey: "user", I18nKey: "admin/user",
	}
	if n != want {
		t.Errorf("NewNamespacedName(Admin::User)\n got %+v\nwant %+v", n, want)
	}
}

func TestNameUncountable(t *testing.T) {
	// "Sheep" is uncountable: plural == singular, so route_key gains "_index".
	n := NewName("Sheep")
	if n.Singular != "sheep" || n.Plural != "sheep" {
		t.Fatalf("unexpected sheep singular/plural: %q/%q", n.Singular, n.Plural)
	}
	if n.RouteKey != "sheep_index" {
		t.Errorf("uncountable route_key = %q, want sheep_index", n.RouteKey)
	}
}

func TestHumanAttributeName(t *testing.T) {
	if got := humanAttributeName("first_name"); got != "First name" {
		t.Errorf("humanAttributeName(first_name) = %q", got)
	}
	if got := humanAttributeName("address.street"); got != "Address street" {
		t.Errorf("humanAttributeName(address.street) = %q", got)
	}
}
