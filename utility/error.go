package utility

import (
	"fmt"
	"strings"
)

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

func GetSQLErr(err error) string {
	errDef := strings.Split(err.Error(), ":")
	errSubstring := errDef[1:]
	switch errDef[0] {
	case "Error 1062":
		return strings.Join(errSubstring, " ")
	case "Error 1366":
		return strings.Join(errSubstring, " ")
	case "Error 3819":
		return "Negative balance violation!"
	default:
		return err.Error()
	}
}
