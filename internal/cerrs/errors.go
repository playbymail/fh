// Package cerrs implements constant errors.
package cerrs

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrNotImplemented = Error("not implemented")
)