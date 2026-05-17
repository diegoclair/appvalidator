package apperrmap_test

import (
	"context"
	"errors"
	"testing"

	"github.com/diegoclair/apperr"
	"github.com/diegoclair/appvalidator"
	"github.com/diegoclair/appvalidator/apperrmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAppErr_Nil(t *testing.T) {
	assert.NoError(t, apperrmap.ToAppErr(nil))
}

func TestToAppErr_PassThroughOnNonValidation(t *testing.T) {
	original := errors.New("some other error")
	got := apperrmap.ToAppErr(original)
	assert.Same(t, original, got)
}

func TestToAppErr_ConvertsValidationError(t *testing.T) {
	ve := &appvalidator.ValidationError{
		Fields: []appvalidator.FieldError{
			{Field: "Email", Tag: "required", Message: "The field 'Email' is required"},
		},
	}

	got := apperrmap.ToAppErr(ve)
	require.Error(t, got)

	var ae apperr.AppError
	require.True(t, errors.As(got, &ae), "expected apperr.AppError, got %T", got)
	assert.Equal(t, apperr.KindValidation, ae.Kind())
	assert.Equal(t, apperr.Code("VALIDATION_ERROR"), ae.Code())

	assert.True(t, errors.Is(got, apperr.ErrValidation))

	var asVE *appvalidator.ValidationError
	require.True(t, errors.As(got, &asVE), "cause chain should preserve *ValidationError")
	assert.Equal(t, ve.Fields, asVE.Fields)

	apErr, ok := got.(*apperr.Error)
	require.True(t, ok)
	meta := apErr.Meta()
	require.NotNil(t, meta)
	fields, ok := meta["fields"].([]appvalidator.FieldError)
	require.True(t, ok, "meta[fields] should be []FieldError")
	assert.Len(t, fields, 1)
	assert.Equal(t, "Email", fields[0].Field)
}

func TestToAppErr_FindsWrappedValidationError(t *testing.T) {
	ve := &appvalidator.ValidationError{
		Fields: []appvalidator.FieldError{{Field: "Name", Tag: "required"}},
	}
	wrapped := errors.Join(errors.New("ctx"), ve)

	got := apperrmap.ToAppErr(wrapped)
	assert.True(t, errors.Is(got, apperr.ErrValidation))
}

func TestNew_ValidateStructReturnsAppErr(t *testing.T) {
	v, err := apperrmap.New()
	require.NoError(t, err)

	data := struct {
		Name string `validate:"required"`
	}{Name: ""}

	got := v.ValidateStruct(context.Background(), data)
	require.Error(t, got)

	assert.True(t, errors.Is(got, apperr.ErrValidation))

	var ae apperr.AppError
	require.True(t, errors.As(got, &ae))
	assert.Equal(t, apperr.KindValidation, ae.Kind())
}

func TestNew_ValidStructReturnsNil(t *testing.T) {
	v, err := apperrmap.New()
	require.NoError(t, err)

	data := struct {
		Name string `validate:"required"`
	}{Name: "John"}

	assert.NoError(t, v.ValidateStruct(context.Background(), data))
}

func TestWrap_UsesInnerValidator(t *testing.T) {
	inner, err := appvalidator.New()
	require.NoError(t, err)

	v := apperrmap.Wrap(inner)

	data := struct {
		CPF string `validate:"cpf"`
	}{CPF: "invalid"}

	got := v.ValidateStruct(context.Background(), data)
	require.Error(t, got)
	assert.True(t, errors.Is(got, apperr.ErrValidation))
}
