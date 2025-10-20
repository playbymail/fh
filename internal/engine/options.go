package engine

import "github.com/playbymail/fh/internal/cerrs"

// Option updates engine settings
type Option func(e *Engine) error

func WithDebugLog() Option {
	return func(e *Engine) error {
		return cerrs.ErrNotImplemented
	}
}
