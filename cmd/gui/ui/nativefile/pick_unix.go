//go:build linux || freebsd || openbsd || netbsd

package nativefile

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func pickNativePath(title string) (string, error) {
	if p, err := runZenity(title); err == nil {
		return p, nil
	} else if errors.Is(err, errCancelled) {
		return "", errCancelled
	}
	if p, err := runKdialog(title); err == nil {
		return p, nil
	} else if errors.Is(err, errCancelled) {
		return "", errCancelled
	}
	return "", errUseFyne
}

func runZenity(title string) (string, error) {
	bin, err := exec.LookPath("zenity")
	if err != nil {
		return "", errUseFyne
	}
	cmd := exec.Command(bin, "--file-selection", "--modal", "--title="+title)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return "", errCancelled
		}
		return "", errUseFyne
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", errCancelled
	}
	return s, nil
}

func runKdialog(title string) (string, error) {
	bin, err := exec.LookPath("kdialog")
	if err != nil {
		return "", errUseFyne
	}
	start := "."
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		start = home
	}
	cmd := exec.Command(bin, "--title", title, "--getopenfilename", start, "*")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return "", errCancelled
		}
		return "", errUseFyne
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", errCancelled
	}
	return s, nil
}
