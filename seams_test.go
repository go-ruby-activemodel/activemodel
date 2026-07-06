// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

import "testing"

func TestTruthy(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{nil, false},
		{false, false},
		{true, true},
		{0, true},
		{"", true},
		{Symbol("x"), true},
	}
	for _, c := range cases {
		if got := truthy(c.v); got != c.want {
			t.Errorf("truthy(%#v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestBlankPresent(t *testing.T) {
	if !isBlank("") || !isBlank(nil) || !isBlank("  ") {
		t.Error("blank values not reported blank")
	}
	if isBlank("x") || isBlank(0) {
		t.Error("present values reported blank")
	}
	if isPresent("") || !isPresent("x") {
		t.Error("isPresent wrong")
	}
}

func TestRubyString(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{nil, ""},
		{"hi", "hi"},
		{Symbol("sym"), "sym"},
		{42, "42"},
		{true, "true"},
	}
	for _, c := range cases {
		if got := rubyString(c.v); got != c.want {
			t.Errorf("rubyString(%#v) = %q, want %q", c.v, got, c.want)
		}
	}
}
