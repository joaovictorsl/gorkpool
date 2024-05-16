package gorkpool

import "fmt"

type ErrIdConflict struct {
	id any
}

func NewErrIdConflict(id any) ErrIdConflict {
	return ErrIdConflict{
		id: id,
	}
}

func (err ErrIdConflict) Error() string {
	return fmt.Sprintf("worker id conflict: there's already a worker with id %v", err.id)
}
