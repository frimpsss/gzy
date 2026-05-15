package sshkeys

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Key struct {
	Name        string
	PrivatePath string
	PublicPath  string
	Fingerprint string
}

type Manager struct {
	SSHKeygenPath string
}

func Discover(sshDir string) ([]Key, error) {
	patterns := []string{"id_ed25519", "id_rsa", "github_*", "gzy_*"}
	seen := map[string]bool{}
	var keys []Key
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(sshDir, pattern))
		if err != nil {
			return nil, err
		}
		for _, privatePath := range matches {
			if strings.HasSuffix(privatePath, ".pub") || seen[privatePath] {
				continue
			}
			publicPath := privatePath + ".pub"
			if _, err := os.Stat(publicPath); err != nil {
				continue
			}
			seen[privatePath] = true
			keys = append(keys, Key{
				Name:        filepath.Base(privatePath),
				PrivatePath: privatePath,
				PublicPath:  publicPath,
			})
		}
	}
	return keys, nil
}

func ReadPublicKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m Manager) Create(privatePath string, email string) error {
	if err := os.MkdirAll(filepath.Dir(privatePath), 0o700); err != nil {
		return err
	}
	keygen := m.SSHKeygenPath
	if keygen == "" {
		keygen = "ssh-keygen"
	}
	cmd := exec.Command(keygen, "-t", "ed25519", "-C", email, "-f", privatePath, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
