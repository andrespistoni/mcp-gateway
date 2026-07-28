package diagnostics

// Check is a redaction-safe diagnostic outcome. Detail is deliberately a
// caller-provided safe summary, never an underlying error or payload.
type Check struct {
	Name   string
	Detail string
	Err    error
}

func Passed(name, detail string) Check { return Check{Name: name, Detail: detail} }

func Failed(name, detail string, err error) Check {
	return Check{Name: name, Detail: detail, Err: err}
}

func (c Check) OK() bool { return c.Err == nil }
