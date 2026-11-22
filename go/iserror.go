package chiklang

import "errors"

// IsDiagnosticError reports whether err carries declaration diagnostics. A
// caller that prints a generic prefix checks this first, because a rendered
// diagnostic already begins with its severity and code.
func IsDiagnosticError(err error) bool {
	var target *Error
	return errors.As(err, &target)
}
