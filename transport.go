package dilithium

import "io"

// Transport abstracts the underlying communication mechanism that dilithium is managing. Implementations are free to
// manage whatever state is necessary, and need to provide a basic `Read`, `Write`, and `Close` facility.
//
type Transport interface {
	io.Reader
	io.Writer
	io.Closer
}
