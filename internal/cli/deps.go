package cli

import "github.com/frimpsss/gzy/internal/config"

type App interface {
	Init() error
	Add() error
	List() ([]config.Account, error)
	Remove(alias string) error
	Install() error
	Export() ([]byte, error)
	Import(path string) error
	Auth(alias string) error
	Doctor() ([]string, error)
	RunGit(alias string, args []string) (int, error)
	Reset(yes bool, deleteKeys bool, deleteGitHub bool) error
	Credential(alias string) (username string, token string, err error)
}
