package injectgo

import "io"

// Initializable is optional to implement.
type Initializable interface {
	Init() error
}

// Closable is optional to implement.
type Closable interface {
	io.Closer
}
