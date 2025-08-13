package errorx

import (
	"errors"
)

type Persistable struct {
	Err error
}

func (e *Persistable) Error() string { return e.Err.Error() }
func (e *Persistable) Unwrap() error { return e.Err }

func NewPersistable(err error) *Persistable {
	if err == nil {
		return nil
	}
	return &Persistable{Err: err}
}

func IsPersistable(err error) bool {
	if err == nil {
		return false
	}

	var p *Persistable
	if errors.As(err, &p) {
		return true
	}

	return false
}
