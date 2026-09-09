package model

import "fmt"

// ExitCodeError represents an error that carries a specific CLI exit code.
type ExitCodeError struct {
	Code int
	Err  error
}

func (e *ExitCodeError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitCodeError) Unwrap() error {
	return e.Err
}

// NewInputError creates an ExitCodeError with exit code 2 (User Input error).
func NewInputError(format string, a ...interface{}) *ExitCodeError {
	return &ExitCodeError{
		Code: 2,
		Err:  fmt.Errorf(format, a...),
	}
}

// NewRuntimeError creates an ExitCodeError with exit code 1 (Git/Runtime error).
func NewRuntimeError(format string, a ...interface{}) *ExitCodeError {
	return &ExitCodeError{
		Code: 1,
		Err:  fmt.Errorf(format, a...),
	}
}
