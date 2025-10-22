// Package cerrs implements constant errors.
package cerrs

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrNotImplemented      = Error("not implemented")
	ErrExists              = Error("already exists")
	ErrNotExist            = Error("does not exist")
	ErrNotOpened           = Error("failed to open")
	ErrSchemaSetupFailed   = Error("schema setup failed")
	ErrSchemaUpgradeFailed = Error("schema upgrade failed")
	ErrSchemaTooNew        = Error("schema version is too new")
	ErrSchemaTooOld        = Error("schema version is too old")
	ErrNoMigrations        = Error("no migrations found")
)
