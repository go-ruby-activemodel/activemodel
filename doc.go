// Copyright (c) the go-ruby-activemodel/activemodel authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package activemodel is a pure-Go (no cgo), MRI-faithful reimplementation of
// Ruby on Rails' ActiveModel, targeting the observable behaviour of the
// activemodel gem on MRI 4.0.5.
//
// This v0.1 foundation ships the Validations + Errors + Naming core — the part
// of ActiveModel that a model object leans on to describe itself and to report
// why it is invalid:
//
//   - Naming (Name)      — ActiveModel::Naming / ActiveModel::Name: derive the
//     singular/plural/element/human/collection/param_key/route_key/i18n_key of a
//     model class name via go-ruby-activesupport's Inflector, byte-for-byte with
//     Rails.
//   - Errors, Error      — ActiveModel::Errors / ActiveModel::Error: add,
//     lookup, where/added?/include?, messages/details, and byte-faithful
//     full_messages ("Name can't be blank") with %{...} interpolation and the
//     complete default-message table.
//   - Validations        — the ActiveModel::Validations engine: the standard
//     validators (presence, absence, length, format, inclusion, exclusion,
//     numericality, confirmation, acceptance), custom validate blocks,
//     validates_each, conditional if/unless/on, and allow_nil/allow_blank, with
//     the exact Rails error types and messages.
//   - Validator, EachValidator — the base contracts for writing custom
//     validators, mirroring ActiveModel::Validator / EachValidator.
//
// Because Go has no open classes, the model object is reached through two small
// seams — Attr (attribute get/set) and Dispatcher (call a method / test respond
// to) — combined in the Model interface. A host such as go-embedded-ruby plugs
// its own object behind them; the tests plug a trivial map-backed fake.
//
// See the README for the roadmap covering the deferred subsystems (Attributes
// typecasting, Dirty tracking, Callbacks, Serialization, SecurePassword, and
// Translation/i18n integration).
package activemodel

// Version is the module version.
const Version = "0.1.0"
