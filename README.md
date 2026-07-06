<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-activemodel/brand/main/social/go-ruby-activemodel-activemodel.png" alt="go-ruby-activemodel/activemodel" width="720"></p>

# activemodel — go-ruby-activemodel

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-activemodel.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby on Rails'
[`ActiveModel`](https://api.rubyonrails.org/classes/ActiveModel.html)** — the
model-behaviour toolkit that gives a plain object validations, an errors
collection, and conventional naming. This v0.1 foundation mirrors the observable
behaviour of the `activemodel` gem on MRI 4.0.5, **without any Ruby runtime**.

It is the ActiveModel backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-activesupport](https://github.com/go-ruby-activesupport/activesupport)
(whose `Inflector` it reuses for naming) and
[go-ruby-set](https://github.com/go-ruby-set/set).

## v0.1 scope — the Validations + Errors + Naming core

| Subsystem | Rails class | What ships |
|-----------|-------------|------------|
| **Naming** | `ActiveModel::Naming` / `Name` | `singular` / `plural` / `element` / `human` / `collection` / `param_key` / `route_key` / `singular_route_key` / `i18n_key`, namespaced and uncountable forms — byte-for-byte with Rails via the ActiveSupport Inflector. |
| **Errors** | `ActiveModel::Errors` / `Error` | `Add`, `[]` (`Get`), `Where`, `Added`, `OfKind`, `Include`, `MessagesFor`, `FullMessagesFor`, `FullMessages`, `Messages`, `Details`, `Clear`, `Size`; `%{...}` interpolation and the full default-message table; byte-faithful full messages ("Name can't be blank"). |
| **Validations** | `ActiveModel::Validations` | the standard validators (**presence, absence, length, format, inclusion, exclusion, numericality, confirmation, acceptance**), custom `Validate` blocks, `ValidatesEach`, conditional `If`/`Unless`/`On`, and `AllowNil`/`AllowBlank` — with the exact Rails error types and messages. |
| **Validator base** | `ActiveModel::Validator` / `EachValidator` | the `Validator` / `EachValidator` interfaces plus `ValidatesWith` / `ValidatesEachWith` for custom validators. |

## The seams — reaching a model without open classes

Go has no open classes, so the object under validation is reached through two
small interfaces (combined as `Model`), which a host such as go-embedded-ruby
plugs its own object behind:

```go
// Attribute get/set — Ruby attr readers/writers and read_attribute_for_validation.
type Attr interface {
	Get(name string) any
	Set(name string, val any)
}

// Method-call seam — a symbol if:/unless: condition, and the respond_to? guard.
type Dispatcher interface {
	Call(method string) any
	RespondTo(method string) bool
}

type Model interface { Attr; Dispatcher }
```

Everything the validators need is expressed through these: `presence` reads the
attribute (`Get`); `confirmation` reads `#{attr}_confirmation` (`Get`); a symbol
`if:` condition calls a method (`Call`); the `%{value}` interpolation honours the
`respond_to?` guard (`RespondTo`). Custom `validate` bodies and proc conditions
are plain Go funcs — the seam for the Ruby block.

## Install

```sh
go get github.com/go-ruby-activemodel/activemodel
```

## Usage

```go
package main

import (
	"fmt"
	"regexp"

	am "github.com/go-ruby-activemodel/activemodel"
)

func main() {
	v := am.New(am.NewName("Person"))
	v.Validates([]string{"name"}, am.Options{Presence: true})
	v.Validates([]string{"email"}, am.Options{
		Format:     &am.FormatOptions{With: regexp.MustCompile(`@`)},
		AllowBlank: true,
	})
	v.Validates([]string{"age"}, am.Options{
		Numericality: &am.NumericalityOptions{OnlyInteger: true, GreaterThan: 0},
	})

	person := /* your Model seam */ nil
	if ok, errs := v.Valid(person); !ok {
		for _, m := range errs.FullMessages() {
			fmt.Println(m) // e.g. "Name can't be blank"
		}
	}

	n := am.NewName("Admin::User")
	fmt.Println(n.Plural, n.RouteKey, n.Human) // admin_users admin_users User
}
```

## Fidelity — the MRI oracle

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
**100%**, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential ActiveModel oracle**: `model_name` fields and `full_messages`
across the whole validator matrix are computed here and diffed against the real
`activemodel` gem running on the system `ruby`. CI installs the gem on the
ubuntu/macos lanes; the oracle skips itself where ruby or the gem is absent.

## Roadmap — the deferred ActiveModel subsystems

v0.1 is the model-description core. Still to come, in rough priority order:

- **Attributes** — the typecasting system (`attribute :born_on, :date`), with the
  type registry and default/coercion behaviour.
- **Dirty** — change tracking (`changed?`, `*_was`, `changes`, `saved_changes`).
- **Callbacks** — `before_validation` / `after_validation` (and the general
  `ActiveModel::Callbacks` define/run machinery) — surfaced as an ordering seam
  around `Valid`.
- **Serialization** — `serializable_hash`, `to_json` / `from_json`, `to_xml`.
- **SecurePassword** — `has_secure_password`, later reusing
  [go-ruby-bcrypt](https://github.com/go-ruby-bcrypt/bcrypt).
- **Translation / I18n** — full `errors.messages` / `activemodel.models` lookup
  chains, locale fallbacks and pluralization backends (v0.1 ships the English
  default table and count-based length pluralization).

## Tests & coverage

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, `gofmt` + `go vet` clean, and green across the six 64-bit Go targets
(amd64, arm64, riscv64, loong64, ppc64le, s390x) and three OSes (Linux, macOS,
Windows). The one dependency is the pure-Go
[go-ruby-activesupport](https://github.com/go-ruby-activesupport/activesupport)
(Inflector).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-activemodel/activemodel authors.
