//go:build !windows && !darwin && !linux && !freebsd && !openbsd && !netbsd

package nativefile

func pickNativePath(title string) (string, error) {
	_ = title
	return "", errUseFyne
}
