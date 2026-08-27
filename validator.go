// Package appvalidator wraps go-playground/validator with custom tags and
// returns structured validation errors that are transport-agnostic.
//
// The core package has zero dependency on apperr — it returns a
// *ValidationError with per-field details. To convert into an apperr.Error,
// use the apperrmap sub-package.
package appvalidator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/klassmann/cpfcnpj"
)

// Validator exposes the validation API.
//
// ValidateStruct returns:
//   - nil when the data set is valid;
//   - *ValidationError when one or more fields fail validation;
//   - a plain error (wrapping validator.InvalidValidationError) when the
//     input itself is invalid (e.g. nil pointer, non-struct).
//
// Custom tags registered by NewValidator:
//   - cpf            — valid Brazilian CPF (uses klassmann/cpfcnpj).
//   - cnpj           — valid Brazilian CNPJ (uses klassmann/cpfcnpj).
//   - required_trim  — required after trimming whitespace (strings only).
type Validator interface {
	ValidateStruct(ctx context.Context, dataSet any) error

	// Pass-through helpers from go-playground/validator/v10.
	Var(field any, tag string) error
	RegisterValidation(tag string, fn validator.Func) error
	RegisterAlias(alias string, tags string)
	StructExcept(current any, fields ...string) error
	StructPartial(current any, fields ...string) error
	StructFiltered(current any, filter validator.FilterFunc) error
}

type validatorImpl struct {
	validator *validator.Validate
}

// New returns a Validator with the custom tags pre-registered.
func New() (Validator, error) {
	v := &validatorImpl{
		// WithRequiredStructEnabled will be the default on go-playground/validator v11.
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}

	v.validator.RegisterTagNameFunc(jsonTagName)

	if err := v.registerCustomValidations(); err != nil {
		return nil, err
	}

	return v, nil
}

func jsonTagName(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	// "-" must fall back to the Go name rather than hide the field from the caller.
	if name == "-" {
		return ""
	}
	return name
}

func (v *validatorImpl) ValidateStruct(ctx context.Context, dataSet any) error {
	err := v.validator.StructCtx(ctx, dataSet)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return fmt.Errorf("appvalidator: invalid argument passed to ValidateStruct: %w", err)
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}

	fields := make([]FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, FieldError{
			Field:       fe.Field(),
			StructField: fe.StructField(),
			Tag:         fe.Tag(),
			Param:       fe.Param(),
			Message:     buildMessage(fe.Field(), fe.Tag(), fe.Param()),
		})
	}

	return &ValidationError{Fields: fields}
}

func buildMessage(field, tag, param string) string {
	switch tag {
	case "required", "required_trim":
		return fmt.Sprintf("The field '%s' is required", field)
	case "email":
		return fmt.Sprintf("The field '%s' should be a valid email", field)
	case "eq":
		return fmt.Sprintf("The value '%s' should be equal to the %s", field, param)
	case "eqfield":
		return fmt.Sprintf("The field '%s' should be equal to the field %s", field, param)
	case "ne":
		return fmt.Sprintf("The value '%s' should not be equal to the %s", field, param)
	case "gte":
		return fmt.Sprintf("The field '%s' should be greater than or equal %s", field, param)
	case "gt":
		return fmt.Sprintf("The field '%s' should be greater than %s", field, param)
	case "lte":
		return fmt.Sprintf("The field '%s' should be less than or equal %s", field, param)
	case "lt":
		return fmt.Sprintf("The field '%s' should be less than %s", field, param)
	case "max":
		return fmt.Sprintf("The field '%s' should have the max length or value: %s", field, param)
	case "min":
		return fmt.Sprintf("The field '%s' should have the minimum length or value: %s", field, param)
	case "uuid4":
		return fmt.Sprintf("The field '%s' should be a valid uuid4", field)
	case "cpf":
		return fmt.Sprintf("The field '%s' should be a valid cpf", field)
	case "cnpj":
		return fmt.Sprintf("The field '%s' should be a valid cnpj", field)
	default:
		return fmt.Sprintf("The field '%s' is invalid", field)
	}
}

func (v *validatorImpl) registerCustomValidations() error {
	if err := v.validator.RegisterValidation("cpf", func(fl validator.FieldLevel) bool {
		cpf := cpfcnpj.NewCPF(fl.Field().String())
		return cpf.IsValid()
	}); err != nil {
		return fmt.Errorf("appvalidator: register cpf: %w", err)
	}

	if err := v.validator.RegisterValidation("cnpj", func(fl validator.FieldLevel) bool {
		cnpj := cpfcnpj.NewCNPJ(fl.Field().String())
		return cnpj.IsValid()
	}); err != nil {
		return fmt.Errorf("appvalidator: register cnpj: %w", err)
	}

	if err := v.validator.RegisterValidation("required_trim", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() != reflect.String {
			return false
		}
		trimmed := strings.TrimSpace(fl.Field().String())
		return v.validator.Var(trimmed, "required") == nil
	}); err != nil {
		return fmt.Errorf("appvalidator: register required_trim: %w", err)
	}

	return nil
}

func (v *validatorImpl) Var(field any, tag string) error {
	return v.validator.Var(field, tag)
}

func (v *validatorImpl) RegisterValidation(tag string, fn validator.Func) error {
	return v.validator.RegisterValidation(tag, fn)
}

func (v *validatorImpl) RegisterAlias(alias string, tags string) {
	v.validator.RegisterAlias(alias, tags)
}

func (v *validatorImpl) StructExcept(current any, fields ...string) error {
	return v.validator.StructExcept(current, fields...)
}

func (v *validatorImpl) StructPartial(current any, fields ...string) error {
	return v.validator.StructPartial(current, fields...)
}

func (v *validatorImpl) StructFiltered(current any, filter validator.FilterFunc) error {
	return v.validator.StructFiltered(current, filter)
}
