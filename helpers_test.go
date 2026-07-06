// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

package activemodel

// fakeModel is the map-backed Model seam the tests plug in place of a Ruby
// object: attrs answer Get/Set, methods answer Call, and RespondTo defaults to
// "responds to any known attribute or method" unless overridden.
type fakeModel struct {
	attrs   map[string]any
	methods map[string]any
	respond map[string]bool
}

func newModel() *fakeModel {
	return &fakeModel{
		attrs:   map[string]any{},
		methods: map[string]any{},
		respond: map[string]bool{},
	}
}

func (f *fakeModel) with(name string, val any) *fakeModel {
	f.attrs[name] = val
	return f
}

func (f *fakeModel) Get(name string) any { return f.attrs[name] }

func (f *fakeModel) Set(name string, val any) { f.attrs[name] = val }

func (f *fakeModel) Call(method string) any { return f.methods[method] }

func (f *fakeModel) RespondTo(method string) bool {
	if v, ok := f.respond[method]; ok {
		return v
	}
	if _, ok := f.attrs[method]; ok {
		return true
	}
	_, ok := f.methods[method]
	return ok
}

func intp(n int) *int { return &n }

func boolp(b bool) *bool { return &b }
