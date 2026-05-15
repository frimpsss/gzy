package tokens

import (
	"os"
	"path/filepath"
	"strings"
)

func Save(dir string, alias string, token string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, alias), []byte(token), 0o600)
}

func Load(dir string, alias string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, alias))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
