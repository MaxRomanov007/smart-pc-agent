package userScope

import (
	"fmt"
	"os"
	"path/filepath"
)

var AppName = "SmartPCAgent"

type CachePath string

func NewCachePath(path string) (CachePath, error) {
	const op = "user-scope.NewCachePath"

	ucd, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("%s: can not get user cache dir: %w", op, err)
	}

	return CachePath(filepath.Join(ucd, AppName, path)), nil
}

func (p *CachePath) SetValue(s string) error {
	cp, err := NewCachePath(s)
	if err != nil {
		return err
	}

	*p = cp
	return nil
}
