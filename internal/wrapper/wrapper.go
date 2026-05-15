package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func RenderUnix(alias, gzyPath string) string {
	return fmt.Sprintf("#!/bin/sh\nexec %s run %s -- \"$@\"\n", strconv.Quote(gzyPath), strconv.Quote(alias))
}

func RenderWindowsCMD(alias, gzyPath string) string {
	return fmt.Sprintf("@echo off\r\n\"%s\" run \"%s\" -- %%*\r\n", gzyPath, alias)
}

func Install(goos, binDir, alias, gzyPath string) (string, error) {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}

	path := wrapperPath(goos, binDir, alias)
	contents := RenderUnix(alias, gzyPath)
	mode := os.FileMode(0755)
	if goos == "windows" {
		contents = RenderWindowsCMD(alias, gzyPath)
		mode = 0644
	}

	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		return "", err
	}
	return path, nil
}

func Remove(goos, binDir, alias string) error {
	err := os.Remove(wrapperPath(goos, binDir, alias))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func wrapperPath(goos, binDir, alias string) string {
	name := "git-" + alias
	if goos == "windows" {
		name += ".cmd"
	}
	return filepath.Join(binDir, name)
}
