package appvalidator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	v, err := New()
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestValidateStruct_Valid(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	data := struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}{Name: "John Doe", Email: "email@test.com"}

	assert.NoError(t, v.ValidateStruct(context.Background(), data))
}

func TestValidateStruct_FieldErrors(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	tests := []struct {
		name     string
		dataSet  any
		wantTag  string
		wantMsg  string
		wantParm string
	}{
		{
			name: "required",
			dataSet: struct {
				Name string `validate:"required"`
			}{Name: ""},
			wantTag: "required",
			wantMsg: "The field 'Name' is required",
		},
		{
			name: "email",
			dataSet: struct {
				Email string `validate:"email"`
			}{Email: "not-email"},
			wantTag: "email",
			wantMsg: "The field 'Email' should be a valid email",
		},
		{
			name: "eq",
			dataSet: struct {
				Age int `validate:"eq=18"`
			}{Age: 17},
			wantTag:  "eq",
			wantParm: "18",
			wantMsg:  "The value 'Age' should be equal to the 18",
		},
		{
			name: "eqfield",
			dataSet: struct {
				Age  int `validate:"eqfield=Name"`
				Name int
			}{Age: 17, Name: 18},
			wantTag:  "eqfield",
			wantParm: "Name",
			wantMsg:  "The field 'Age' should be equal to the field Name",
		},
		{
			name: "ne",
			dataSet: struct {
				Age int `validate:"ne=18"`
			}{Age: 18},
			wantTag:  "ne",
			wantParm: "18",
			wantMsg:  "The value 'Age' should not be equal to the 18",
		},
		{
			name: "gte",
			dataSet: struct {
				Age int `validate:"gte=18"`
			}{Age: 17},
			wantTag:  "gte",
			wantParm: "18",
			wantMsg:  "The field 'Age' should be greater than or equal 18",
		},
		{
			name: "gt",
			dataSet: struct {
				Age int `validate:"gt=18"`
			}{Age: 18},
			wantTag:  "gt",
			wantParm: "18",
			wantMsg:  "The field 'Age' should be greater than 18",
		},
		{
			name: "lte",
			dataSet: struct {
				Age int `validate:"lte=18"`
			}{Age: 19},
			wantTag:  "lte",
			wantParm: "18",
			wantMsg:  "The field 'Age' should be less than or equal 18",
		},
		{
			name: "lt",
			dataSet: struct {
				Age int `validate:"lt=18"`
			}{Age: 18},
			wantTag:  "lt",
			wantParm: "18",
			wantMsg:  "The field 'Age' should be less than 18",
		},
		{
			name: "max",
			dataSet: struct {
				Age int `validate:"max=18"`
			}{Age: 19},
			wantTag:  "max",
			wantParm: "18",
			wantMsg:  "The field 'Age' should have the max length or value: 18",
		},
		{
			name: "min",
			dataSet: struct {
				Age int `validate:"min=18"`
			}{Age: 17},
			wantTag:  "min",
			wantParm: "18",
			wantMsg:  "The field 'Age' should have the minimum length or value: 18",
		},
		{
			name: "uuid4",
			dataSet: struct {
				UUID string `validate:"uuid4"`
			}{UUID: "123"},
			wantTag: "uuid4",
			wantMsg: "The field 'UUID' should be a valid uuid4",
		},
		{
			name: "cpf invalid",
			dataSet: struct {
				CPF string `validate:"cpf"`
			}{CPF: "12345678910"},
			wantTag: "cpf",
			wantMsg: "The field 'CPF' should be a valid cpf",
		},
		{
			name: "cnpj invalid",
			dataSet: struct {
				CNPJ string `validate:"cnpj"`
			}{CNPJ: "1234567891011"},
			wantTag: "cnpj",
			wantMsg: "The field 'CNPJ' should be a valid cnpj",
		},
		{
			name: "required_trim with only spaces",
			dataSet: struct {
				Name string `validate:"required_trim"`
			}{Name: "   "},
			wantTag: "required_trim",
			wantMsg: "The field 'Name' is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateStruct(context.Background(), tt.dataSet)
			require.Error(t, err)

			var ve *ValidationError
			require.True(t, errors.As(err, &ve), "expected *ValidationError, got %T", err)
			require.Len(t, ve.Fields, 1)

			assert.Equal(t, tt.wantTag, ve.Fields[0].Tag)
			assert.Equal(t, tt.wantMsg, ve.Fields[0].Message)
			if tt.wantParm != "" {
				assert.Equal(t, tt.wantParm, ve.Fields[0].Param)
			}
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestValidateStruct_MultipleErrors(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	data := struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=18"`
	}{Name: "", Email: "bad", Age: 10}

	err = v.ValidateStruct(context.Background(), data)
	require.Error(t, err)

	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Len(t, ve.Fields, 3)
}

func TestValidateStruct_NonStruct(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	err = v.ValidateStruct(context.Background(), "not a struct")
	require.Error(t, err)

	var ve *ValidationError
	assert.False(t, errors.As(err, &ve), "non-struct input should not yield ValidationError")
}

func TestValidateStruct_CustomTag_Valid(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	data := struct {
		Name string `validate:"required_trim"`
	}{Name: "  ok  "}

	assert.NoError(t, v.ValidateStruct(context.Background(), data))
}
