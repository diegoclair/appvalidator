# appvalidator

<p align="center">
  <b>Struct validation for Go with structured, transport-agnostic errors</b><br>
  Wraps <code>go-playground/validator/v10</code>, adds business-friendly tags, and returns a
  typed <code>*ValidationError</code> ready to be mapped to any transport.
  <br><br>
  <a href="https://github.com/diegoclair/appvalidator/actions/workflows/ci.yml">
    <img src="https://github.com/diegoclair/appvalidator/actions/workflows/ci.yml/badge.svg" alt="CI" />
  </a>
  <a href="https://github.com/diegoclair/appvalidator/tags">
    <img src="https://img.shields.io/github/tag/diegoclair/appvalidator.svg" alt="GitHub tag" />
  </a>
  <a href="https://pkg.go.dev/github.com/diegoclair/appvalidator">
    <img src="https://pkg.go.dev/badge/github.com/diegoclair/appvalidator.svg" alt="Go Reference" />
  </a>
  <a href="https://goreportcard.com/report/github.com/diegoclair/appvalidator">
    <img src="https://goreportcard.com/badge/github.com/diegoclair/appvalidator" alt="Go Report Card" />
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License" />
  </a>
</p>

## Introduction

### Why

`go-playground/validator` is great but raw — its errors are flat strings, every project ends up reimplementing the same custom tags, friendly messages, and a structured error type the API layer can serialize:

```go
func (s *userService) Signup(ctx context.Context, dto SignupDTO) error {
    if err := validate.Struct(dto); err != nil {
        // ❌ Flat string: "Key: 'SignupDTO.Email' Error:Field validation for 'Email' failed on the 'required' tag"
        // ❌ Frontend has to parse the string to know which field/rule failed
        // ❌ No i18n: messages are hardcoded in English
        // ❌ Every project re-registers `cpf`, `cnpj`, `required_trim`, …
        return err
    }
    // ...
}
```

### How

With `appvalidator`, you get a typed `*ValidationError` with per-field details (`Field`, `Tag`, `Param`, `Message`). The transport layer decides how to render it — HTTP, gRPC, GraphQL:

```go
func (s *userService) Signup(ctx context.Context, dto SignupDTO) error {
    if err := s.validator.ValidateStruct(ctx, dto); err != nil {
        return err                                       // ✅ structured *ValidationError
    }                                                    // ✅ per-field {Field, Tag, Param, Message}
    // ...                                               // ✅ custom tags (cpf, cnpj, required_trim) built-in
}                                                        // ✅ transport-agnostic — no net/http import
```

Pair with [`apperr`](https://github.com/diegoclair/apperr) via the [`apperrmap`](./apperrmap) sub-package and you get end-to-end transport-agnostic errors with i18n-ready codes.

## Install

```bash
go get github.com/diegoclair/appvalidator
```

## Getting Started

### 1. Create the validator

```go
import "github.com/diegoclair/appvalidator"

v, err := appvalidator.New()
if err != nil {
    log.Fatal(err)
}
```

`New()` returns a `Validator` with the custom tags (`cpf`, `cnpj`, `required_trim`) pre-registered.

### 2. Validate your DTOs

```go
type SignupDTO struct {
    Name  string `validate:"required_trim"`
    Email string `validate:"required,email"`
    CPF   string `validate:"required,cpf"`
    Age   int    `validate:"gte=18"`
}

err := v.ValidateStruct(ctx, SignupDTO{
    Email: "not-an-email",
    CPF:   "111",
})
```

All standard `go-playground/validator` tags work too — `required`, `email`, `min`, `max`, `gte`, `lt`, `eqfield`, `uuid4`, etc.

### 3. Handle the structured error

```go
var ve *appvalidator.ValidationError
if errors.As(err, &ve) {
    for _, f := range ve.Fields {
        // f.Field   → "Email"
        // f.Tag     → "required" | "cpf" | "min"
        // f.Param   → "8" for min=8 (empty when no param)
        // f.Message → "The field 'Email' is required"
    }
}
```

`Field`, `Tag`, and `Param` give the frontend everything it needs to translate the error (e.g. `{field: "email", rule: "required"}`) — no string parsing. `Message` is a sane fallback for logs.

### 4. Map to apperr (optional)

The [`apperrmap`](./apperrmap) sub-package converts validation results into [`apperr.Error`](https://github.com/diegoclair/apperr):

```go
import "github.com/diegoclair/appvalidator/apperrmap"

// Option 1 — wrap the validator (recommended)
v, _ := apperrmap.NewValidator()
err := v.ValidateStruct(ctx, dto) // err is apperr-compatible on failure

// Option 2 — convert on the way out
err := plainValidator.ValidateStruct(ctx, dto)
return apperrmap.ToAppErr(err)
```

The result is `apperr.ErrValidation` (`Kind=Validation`, `Code="VALIDATION_ERROR"`) with `meta["fields"] = []FieldError`. The original `*ValidationError` is kept in the cause chain, so `errors.As` still works.

Combined with [`apperr/httpmap`](https://github.com/diegoclair/apperr/tree/main/httpmap) you get:

```json
{
    "message": "validation error",
    "status_code": 400,
    "error": "Bad Request",
    "code": "VALIDATION_ERROR",
    "meta": {
        "fields": [
            {"field": "Email", "tag": "required", "message": "The field 'Email' is required"},
            {"field": "CPF",   "tag": "cpf",      "message": "The field 'CPF' should be a valid cpf"}
        ]
    }
}
```

The core `appvalidator` package does **not** import `apperr`. Only the `apperrmap` sub-package does — import what you actually use.

## Built-in Custom Tags

| Tag             | Description                                                            |
|-----------------|------------------------------------------------------------------------|
| `cpf`           | Valid Brazilian CPF (via `klassmann/cpfcnpj`).                         |
| `cnpj`          | Valid Brazilian CNPJ (via `klassmann/cpfcnpj`).                        |
| `required_trim` | Required after trimming whitespace — useful for form inputs (strings). |

Need more? Use `RegisterValidation` to plug in your own:

```go
v.RegisterValidation("strong_password", func(fl validator.FieldLevel) bool {
    return len(fl.Field().String()) >= 12
})
```

## Architecture

```
appvalidator (core — zero apperr dependency)
├── Validator           interface
├── ValidationError     structured error (Fields []FieldError)
├── FieldError          {Field, Tag, Param, Message}
└── New()               constructor with custom tags pre-registered

apperrmap/ (sub-package — opt-in apperr bridge)
├── ToAppErr(err)       *ValidationError → apperr.ErrValidation + meta
├── NewValidator()      returns Validator that already converts on failure
└── Wrap(inner)         wraps an existing appvalidator.Validator
```

## Design Decisions

### Why a wrapper instead of using `validator/v10` directly?

The wrapper centralizes three things every project needs:
- **Custom domain tags** (`cpf`, `cnpj`, `required_trim`) — registered once, available everywhere
- **Friendly messages** — `buildMessage` translates raw tags into human-readable text
- **A structured error type** — `*ValidationError` with per-field details instead of a flat string

You still have access to the underlying API (`Var`, `RegisterValidation`, `RegisterAlias`, `StructPartial`, …) through pass-through methods.

### Why a separate `apperrmap` sub-package?

The core package has zero dependency on `apperr`. If you don't use `apperr`, you pay nothing — your `go.sum` stays clean. Projects that do use `apperr` opt-in by importing `apperrmap`.

### Why `WithRequiredStructEnabled`?

This will be the default in `validator/v11`. Enabling it now means structs marked `required` are validated as expected (no silent skip on zero-value structs) and you won't be surprised on the upgrade.

## Roadmap

- [ ] Configurable message templates (i18n via `Tag` + `Param`)
- [ ] Built-in tags for other locales (e.g. EU VAT, SSN)

## Contributing

Contributions are welcome!

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

## License

[MIT](./LICENSE)
