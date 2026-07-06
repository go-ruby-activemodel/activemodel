// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import (
	"strings"

	inf "github.com/go-ruby-activesupport/activesupport/inflector"
)

// Name is ActiveModel::Name: the bundle of conventional names Rails derives from
// a model class name (and an optional namespace module name). Every field is
// computed through go-ruby-activesupport's Inflector, byte-for-byte with Rails'
// ActiveModel::Name#initialize.
type Name struct {
	// Name is the class name as given, e.g. "Person" or "Admin::User".
	Name string
	// Singular is the underscored, "/"-flattened name: "person", "admin_user".
	Singular string
	// Plural pluralizes Singular: "people", "admin_users".
	Plural string
	// Element is the underscored, demodulized name: "person", "user".
	Element string
	// Human is the humanized Element: "Person", "User".
	Human string
	// Collection is the tableized name: "people", "admin/users".
	Collection string
	// ParamKey is the key for params: Singular (or the unnamespaced singular).
	ParamKey string
	// RouteKey is the plural route key ("people"), with "_index" appended for an
	// uncountable name.
	RouteKey string
	// SingularRouteKey singularizes RouteKey.
	SingularRouteKey string
	// I18nKey is the underscored name used as an i18n scope key.
	I18nKey string
}

// NewName builds the Name for a class name with no namespace, mirroring
// ActiveModel::Name.new(klass) where klass.name == name.
func NewName(name string) Name { return NewNamespacedName(name, "") }

// NewNamespacedName builds the Name for a class name nested in a namespace
// module, mirroring ActiveModel::Name.new(klass, namespace). namespace is the
// module's own name (e.g. "Admin"); an empty namespace means none.
func NewNamespacedName(name, namespace string) Name {
	unnamespaced := name
	if namespace != "" {
		unnamespaced = strings.TrimPrefix(name, namespace+"::")
	}

	n := Name{Name: name}
	n.Singular = singularizeName(name)
	n.Plural = inf.Pluralize(n.Singular)
	n.Element = inf.Underscore(inf.Demodulize(name))
	n.Human = inf.Humanize(n.Element)
	n.Collection = inf.Tableize(name)
	n.I18nKey = inf.Underscore(name)

	if namespace != "" {
		n.ParamKey = singularizeName(unnamespaced)
		n.RouteKey = inf.Pluralize(n.ParamKey)
	} else {
		n.ParamKey = n.Singular
		n.RouteKey = n.Plural
	}
	n.SingularRouteKey = inf.Singularize(n.RouteKey)
	if n.Plural == n.Singular { // uncountable
		n.RouteKey += "_index"
	}
	return n
}

// singularizeName is ActiveModel::Name#_singularize: underscore then flatten "/"
// to "_" ("Admin::User" -> "admin/user" -> "admin_user").
func singularizeName(s string) string {
	return strings.ReplaceAll(inf.Underscore(s), "/", "_")
}

// humanAttributeName is ActiveModel's default human_attribute_name: humanize the
// attribute with any dotted path flattened to underscores ("first_name" -> "First
// name", "address.street" -> "Address street").
func humanAttributeName(attribute string) string {
	return inf.Humanize(strings.ReplaceAll(attribute, ".", "_"))
}
