package dilithium

import "io"

type Transport interface {
	io.Reader
	io.Writer
	io.Closer
}
