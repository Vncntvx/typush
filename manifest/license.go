// SPDX license validation for Universe submissions.
// Mirrors typst/packages bundler rules: every license in the expression must
// be OSI-approved or an allowed CC license (any version of CC-BY, CC-BY-SA,
// CC0). LicenseRef-* / referencer ids are rejected.
package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// osiApproved is a pragmatic allowlist of common OSI-approved SPDX ids.
// Anything missing here but genuinely OSI-approved will be reported as an
// error asking the user to file an issue — same spirit as the bundler.
var osiApproved = map[string]bool{
	"0BSD": true, "AFL-3.0": true, "AGPL-3.0-only": true, "AGPL-3.0-or-later": true,
	"Apache-2.0": true, "APL-1.0": true, "APSL-2.0": true, "Artistic-2.0": true,
	"BSD-2-Clause": true, "BSD-3-Clause": true, "BSD-3-Clause-Clear": true,
	"BSL-1.0": true, "CDDL-1.0": true, "CECILL-2.1": true, "CPAL-1.0": true,
	"CPL-1.0": true, "ECL-2.0": true, "EFL-2.0": true, "EPL-1.0": true,
	"EPL-2.0": true, "EUPL-1.1": true, "EUPL-1.2": true, "GPL-2.0-only": true,
	"GPL-2.0-or-later": true, "GPL-3.0-only": true, "GPL-3.0-or-later": true,
	"ISC": true, "LGPL-2.0-only": true, "LGPL-2.0-or-later": true,
	"LGPL-2.1-only": true, "LGPL-2.1-or-later": true, "LGPL-3.0-only": true,
	"LGPL-3.0-or-later": true, "MIT": true, "MIT-0": true, "MPL-2.0": true,
	"MS-PL": true, "MS-RL": true, "NCSA": true, "OFL-1.1": true,
	"OSL-3.0": true, "PostgreSQL": true, "Python-2.0": true, "Ruby": true,
	"Unlicense": true, "UPL-1.0": true, "Vim": true, "WTFPL": true,
	"Zlib": true, "ZPL-2.1": true,
	// CC0 is not OSI-approved but explicitly allowed by Universe.
	"CC0-1.0": true,
}

var ccRe = regexp.MustCompile(`^CC(-BY|-BY-SA|0)-[0-9]\.[0-9](-[A-Z]+)?$`)

// ValidateLicense checks an SPDX-2 license expression per Universe rules.
func ValidateLicense(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return fmt.Errorf("package license is missing")
	}
	tokens := tokenizeSPDX(expr)
	if len(tokens) == 0 {
		return fmt.Errorf("failed to parse SPDX license expression %q", expr)
	}
	parens := 0
	wantOperand := true
	seenLicense := false
	for _, t := range tokens {
		switch t {
		case "(":
			if !wantOperand {
				return fmt.Errorf("failed to parse SPDX license expression %q", expr)
			}
			parens++
		case ")":
			if wantOperand || parens == 0 {
				return fmt.Errorf("failed to parse SPDX license expression %q", expr)
			}
			parens--
		case "AND", "OR":
			if wantOperand {
				return fmt.Errorf("failed to parse SPDX license expression %q", expr)
			}
			wantOperand = true
		case "WITH":
			if wantOperand {
				return fmt.Errorf("failed to parse SPDX license expression %q", expr)
			}
			// WITH <exception>: exception id follows; skip validation of it,
			// but an operand must follow.
			wantOperand = true
		default:
			if !wantOperand {
				return fmt.Errorf("failed to parse SPDX license expression %q", expr)
			}
			// After WITH comes an exception id, not a license.
			// We detect that by checking the previous token.
			if isExceptionPosition(tokens, t) {
				wantOperand = false
				continue
			}
			if err := validateLicenseID(t); err != nil {
				return err
			}
			seenLicense = true
			wantOperand = false
		}
	}
	if parens != 0 || wantOperand || !seenLicense {
		return fmt.Errorf("failed to parse SPDX license expression %q", expr)
	}
	return nil
}

// isExceptionPosition reports whether token t directly follows a WITH.
func isExceptionPosition(tokens []string, t string) bool {
	for i, tok := range tokens {
		if tok == t && i > 0 && tokens[i-1] == "WITH" {
			return true
		}
	}
	return false
}

func validateLicenseID(id string) error {
	base := strings.TrimSuffix(id, "+") // deprecated trailing-+ form
	if strings.HasPrefix(base, "LicenseRef") || strings.Contains(base, ":") {
		return fmt.Errorf("license must not contain a referencer: %q", id)
	}
	if osiApproved[base] || ccRe.MatchString(base) {
		return nil
	}
	return fmt.Errorf("license is neither OSI approved nor an allowed CC license: %q", id)
}

func tokenizeSPDX(expr string) []string {
	expr = strings.ReplaceAll(expr, "(", " ( ")
	expr = strings.ReplaceAll(expr, ")", " ) ")
	var out []string
	for _, f := range strings.Fields(expr) {
		up := strings.ToUpper(f)
		if up == "AND" || up == "OR" || up == "WITH" {
			out = append(out, up)
			continue
		}
		out = append(out, f)
	}
	return out
}
