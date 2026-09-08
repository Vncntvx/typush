// Package lint implements the network-free parts of typst/package-check:
// README, files and import rules. Diagnostics mirror upstream codes.
package checker

// Severity of a diagnostic.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Diag is one lint finding.
type Diag struct {
	Severity Severity
	Code     string
	Message  string
}

func errf(code, msg string) Diag  { return Diag{Severity: Error, Code: code, Message: msg} }
func warnf(code, msg string) Diag { return Diag{Severity: Warning, Code: code, Message: msg} }

// Split separates errors from warnings.
func Split(diags []Diag) (errs, warns []Diag) {
	for _, d := range diags {
		if d.Severity == Error {
			errs = append(errs, d)
		} else {
			warns = append(warns, d)
		}
	}
	return errs, warns
}
