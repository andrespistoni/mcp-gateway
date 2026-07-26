package diagnostics

import "errors"

type Kind string

const (
	Usage         Kind = "usage"
	Validation    Kind = "validation"
	Configuration Kind = "configuration"
	Persistence   Kind = "persistence"
	Process       Kind = "process"
	Network       Kind = "network"
	Protocol      Kind = "protocol"
	Unavailable   Kind = "unavailable"
	Timeout       Kind = "timeout"
	Resource      Kind = "resource"
	Security      Kind = "security"
	Conflict      Kind = "conflict"
	Shutdown      Kind = "shutdown"
)

// Fault conserva la causa para inspección interna, pero expone un mensaje estable.
type Fault struct {
	kind    Kind
	message string
	cause   error
	fields  []Field
}

func NewFault(kind Kind, message string, cause error, fields ...Field) *Fault {
	return &Fault{kind: kind, message: message, cause: cause, fields: append([]Field(nil), fields...)}
}

func (f *Fault) Error() string {
	return f.message
}

func (f *Fault) Unwrap() error {
	return f.cause
}

func (f *Fault) Kind() Kind {
	return f.kind
}

func (f *Fault) Message() string {
	return f.message
}

func (f *Fault) Fields() []Field {
	return append([]Field(nil), f.fields...)
}

func KindOf(err error) Kind {
	var fault *Fault
	if errors.As(err, &fault) {
		return fault.Kind()
	}
	return ""
}

func ExternalMessage(err error) string {
	var fault *Fault
	if errors.As(err, &fault) {
		return fault.Message()
	}
	return "error interno"
}
