package proc

import "io"

type ProcessTree interface {
	// El caller es propietario de los tres handles y debe cerrarlos. Wait solo
	// espera el proceso y nunca cierra un parent pipe que aún pueda drenarse.
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Wait() error
	Terminate() error
	Kill() error
}
