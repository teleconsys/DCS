//go:build windows || darwin

package nativefile

import (
	"errors"

	sq "github.com/sqweek/dialog"
)

func pickNativePath(title string) (string, error) {
	path, err := sq.File().Title(title).Load()
	if err != nil {
		if errors.Is(err, sq.ErrCancelled) {
			return "", errCancelled
		}
		return "", err
	}
	return path, nil
}
