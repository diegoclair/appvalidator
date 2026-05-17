package appvalidator

import "strings"

// ValidationError is the error returned by ValidateStruct when one or more
// fields fail validation. It carries structured per-field details so that
// callers (or mappers like apperrmap) can build their own representation —
// HTTP responses, gRPC status, i18n messages, etc.
type ValidationError struct {
	Fields []FieldError
}

// FieldError describes a single validation failure.
//   - Field is the struct field name (e.g. "Email").
//   - Tag is the validator tag that failed (e.g. "required", "cpf", "min").
//   - Param is the tag parameter when present (e.g. "8" for min=8); empty otherwise.
//   - Message is a human-readable description, ready for display or logging.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

// Error joins all field messages with "; " so the error is useful as a
// fallback string (logs, plain text responses). Structured access is
// available via Fields.
func (v *ValidationError) Error() string {
	if len(v.Fields) == 0 {
		return "validation error"
	}
	msgs := make([]string, 0, len(v.Fields))
	for _, f := range v.Fields {
		msgs = append(msgs, f.Message)
	}
	return strings.Join(msgs, "; ")
}
