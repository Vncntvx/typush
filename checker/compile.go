// Package compile runs the local Typst compiler against a package:
// a smoke import of the library and, for templates, the official
// `typst init` + `compile` flow. Everything runs offline in isolated
// temp HOME/XDG directories with the package symlinked into @preview.
package checker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Result collects compiler findings.
type Result struct {
	Errors   []string
	Warnings []string
}

func (r *Result) addErr(f string, a ...any)  { r.Errors = append(r.Errors, fmt.Sprintf(f, a...)) }
func (r *Result) addWarn(f string, a ...any) { r.Warnings = append(r.Warnings, fmt.Sprintf(f, a...)) }
func (r *Result) merge(o Result) {
	r.Errors = append(r.Errors, o.Errors...)
	r.Warnings = append(r.Warnings, o.Warnings...)
}

// LookPath returns the typst binary (TYPST_BIN overrides PATH).
func LookPath() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("TYPST_BIN")); v != "" {
		return v, true
	}
	p, err := exec.LookPath("typst")
	if err != nil {
		return "", false
	}
	return p, true
}

// TypstVersion extracts the semver string of the typst binary, or "" if unparseable.
func TypstVersion(bin string) string {
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	if len(fields) >= 2 {
		return fields[1]
	}
	return ""
}

// isolatedEnv creates temp HOME/XDG dirs and symlinks pkgDir into
// @preview/<name>/<version> where the child typst process looks.
// Returns env additions and a cleanup func.
func isolatedEnv(pkgDir, name, version string) (env []string, cleanup func(), err error) {
	abs, err := filepath.Abs(pkgDir)
	if err != nil {
		return nil, nil, err
	}
	tmp, err := os.MkdirTemp("", "typush-check-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup = func() { os.RemoveAll(tmp) }
	fail := func(e error) ([]string, func(), error) {
		cleanup()
		return nil, nil, e
	}

	dataBase := filepath.Join(tmp, "data")
	cacheBase := filepath.Join(tmp, "cache")
	var typstData string
	switch runtime.GOOS {
	case "darwin":
		typstData = filepath.Join(tmp, "home", "Library", "Application Support")
	case "windows":
		typstData = filepath.Join(tmp, "appdata")
	default:
		typstData = dataBase
	}
	link := filepath.Join(typstData, "typst", "packages", "preview", name, version)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return fail(err)
	}
	if err := os.Symlink(abs, link); err != nil {
		return fail(fmt.Errorf("failed to symlink package for isolated check: %w", err))
	}
	home := filepath.Join(tmp, "home")
	_ = os.MkdirAll(home, 0o755)
	env = []string{
		"HOME=" + home,
		"XDG_DATA_HOME=" + dataBase,
		"XDG_CACHE_HOME=" + cacheBase,
	}
	if runtime.GOOS == "windows" {
		env = append(env, "APPDATA="+filepath.Join(tmp, "appdata"), "LOCALAPPDATA="+filepath.Join(tmp, "appdata"))
	}
	// Keep PATH so typst and helpers resolve.
	env = append(env, "PATH="+os.Getenv("PATH"))
	return env, cleanup, nil
}

func runTypst(bin string, env []string, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if len(s) > 6000 {
		s = s[:6000] + "\n... (truncated)"
	}
	return s, err
}

// splitWarnings extracts typst warning lines from CLI output.
func splitWarnings(output string) []string {
	var out []string
	for _, line := range strings.Split(output, "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		low := strings.ToLower(l)
		if strings.Contains(low, "unknown font family") {
			continue // same filter as package-check
		}
		if strings.HasPrefix(low, "warning") {
			out = append(out, l)
		}
	}
	return out
}

// CheckLibrary compiles a smoke file importing the package.
func CheckLibrary(bin, pkgDir, name, version string) Result {
	var r Result
	env, cleanup, err := isolatedEnv(pkgDir, name, version)
	if err != nil {
		r.addErr("compile setup failed: %v", err)
		return r
	}
	defer cleanup()

	work, err := os.MkdirTemp("", "typush-smoke-*")
	if err != nil {
		r.addErr("compile setup failed: %v", err)
		return r
	}
	defer os.RemoveAll(work)
	smoke := fmt.Sprintf("#import \"@preview/%s:%s\": *\n", name, version)
	if err := os.WriteFile(filepath.Join(work, "smoke.typ"), []byte(smoke), 0o644); err != nil {
		r.addErr("compile setup failed: %v", err)
		return r
	}
	out, err := runTypst(bin, env, work, "compile", "smoke.typ", "smoke.pdf")
	if err != nil {
		r.addErr("package failed to compile (import check):\n%s", nonEmpty(out, err.Error()))
		return r
	}
	for _, w := range splitWarnings(out) {
		r.addWarn("compiler warning (import check): %s", w)
	}
	return r
}

// CheckTemplate runs `typst init @preview/name:version` then compiles the
// template entrypoint, mirroring the official template test workflow.
func CheckTemplate(bin, pkgDir, name, version, tplEntry string) Result {
	var r Result
	env, cleanup, err := isolatedEnv(pkgDir, name, version)
	if err != nil {
		r.addErr("compile setup failed: %v", err)
		return r
	}
	defer cleanup()

	work, err := os.MkdirTemp("", "typush-tpl-*")
	if err != nil {
		r.addErr("compile setup failed: %v", err)
		return r
	}
	defer os.RemoveAll(work)
	spec := fmt.Sprintf("@preview/%s:%s", name, version)
	if out, err := runTypst(bin, env, work, "init", spec, "proj"); err != nil {
		r.addErr("`typst init %s` failed:\n%s", spec, nonEmpty(out, err.Error()))
		return r
	}
	proj := filepath.Join(work, "proj")
	entry := filepath.Join(proj, filepath.FromSlash(tplEntry))
	if _, err := os.Stat(entry); err != nil {
		r.addErr("template entrypoint %q missing after `typst init`", tplEntry)
		return r
	}
	out, err := runTypst(bin, env, proj, "compile", tplEntry, "out.pdf")
	if err != nil {
		r.addErr("template failed to compile out-of-the-box:\n%s", nonEmpty(out, err.Error()))
		return r
	}
	for _, w := range splitWarnings(out) {
		r.addWarn("compiler warning (template): %s", w)
	}
	return r
}

func nonEmpty(out, fallback string) string {
	if strings.TrimSpace(out) == "" {
		return fallback
	}
	return out
}
