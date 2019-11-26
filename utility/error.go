package utility

import "fmt"

type AppError struct {
	ErrType string
	Err     error
}

func (e AppError) Type() string {
	return fmt.Sprintf("%s", e.ErrType)
}

func (e AppError) Error() string {
	return fmt.Sprintf("%s", e.Err)
}
