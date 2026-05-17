// Package apperrmap bridges appvalidator's *ValidationError into an
// apperr.Error so transport layers using apperr (e.g. apperr/httpmap) get
// a consistent shape — Kind=Validation, code="VALIDATION_ERROR", and the
// field-level details in meta["fields"].
//
// Two ways to use it:
//
//  1. Convert on the way out, keeping the plain appvalidator at the entry:
//
//     err := v.ValidateStruct(ctx, dto)
//     return apperrmap.ToAppErr(err)
//
//  2. Use the wrapping constructor so every ValidateStruct call already
//     returns an apperr-compatible error:
//
//     v, _ := apperrmap.New()
//     err := v.ValidateStruct(ctx, dto) // err is *apperr.Error on failure
package apperrmap

import (
	"context"
	"errors"

	"github.com/diegoclair/apperr"
	"github.com/diegoclair/appvalidator"
)

// ToAppErr converts a validation error produced by appvalidator into an
// apperr.Error.
//
// Behavior:
//   - If err is nil, returns nil.
//   - If err carries a *appvalidator.ValidationError (direct or wrapped),
//     returns apperr.ErrValidation with meta["fields"]=[]FieldError and the
//     original error chained as cause (so errors.As still finds the
//     *ValidationError downstream).
//   - Otherwise, returns err unchanged — non-validation errors are not
//     touched.
func ToAppErr(err error) error {
	if err == nil {
		return nil
	}

	var ve *appvalidator.ValidationError
	if !errors.As(err, &ve) {
		return err
	}

	return apperr.ErrValidation.
		WithMeta("fields", ve.Fields).
		WithCause(err)
}

// Validator mirrors appvalidator.Validator but guarantees that
// ValidateStruct returns an apperr-compatible error on failure.
type Validator interface {
	ValidateStruct(ctx context.Context, dataSet any) error
}

type wrapped struct {
	inner appvalidator.Validator
}

// New returns a Validator that wraps appvalidator.New and converts
// validation failures to apperr.Error transparently.
func New() (Validator, error) {
	inner, err := appvalidator.New()
	if err != nil {
		return nil, err
	}
	return &wrapped{inner: inner}, nil
}

// Wrap turns an existing appvalidator.Validator into an apperr-aware one.
// Useful when the caller already configured custom rules on the inner
// validator (e.g. RegisterValidation) and wants apperr translation on top.
func Wrap(v appvalidator.Validator) Validator {
	return &wrapped{inner: v}
}

func (w *wrapped) ValidateStruct(ctx context.Context, dataSet any) error {
	return ToAppErr(w.inner.ValidateStruct(ctx, dataSet))
}
