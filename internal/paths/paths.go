package paths

import (
	"os"
	"path"
	"runtime"
	"strings"
)

type Paths struct {
	goos string
	env  map[string]string
}

func New(goos string, env map[string]string) Paths {
	copied := make(map[string]string, len(env))
	for key, value := range env {
		copied[key] = value
	}
	return Paths{goos: goos, env: copied}
}

func FromOS() Paths {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return New(runtime.GOOS, env)
}

func (p Paths) ConfigFile() string {
	switch p.goos {
	case "darwin":
		return p.join(p.home(), "Library", "Application Support", "gzy", "config.json")
	case "windows":
		base := p.env["APPDATA"]
		if base == "" {
			base = p.join(p.home(), "AppData", "Roaming")
		}
		return p.join(base, "gzy", "config.json")
	default:
		base := p.env["XDG_CONFIG_HOME"]
		if base == "" {
			base = p.join(p.home(), ".config")
		}
		return p.join(base, "gzy", "config.json")
	}
}

func (p Paths) SSHDir() string {
	return p.join(p.home(), ".ssh")
}

func (p Paths) DefaultKeyPair(alias string) (string, string) {
	privateKey := p.join(p.SSHDir(), "gzy_"+alias)
	return privateKey, privateKey + ".pub"
}

func (p Paths) BinDir() string {
	if p.goos == "windows" {
		return p.join(p.home(), "bin")
	}

	localBin := p.join(p.home(), ".local", "bin")
	if p.pathContains(localBin) {
		return localBin
	}
	return p.join(p.home(), "bin")
}

func (p Paths) home() string {
	if p.goos == "windows" {
		return p.env["USERPROFILE"]
	}
	return p.env["HOME"]
}

func (p Paths) pathContains(dir string) bool {
	normalizedDir := p.normalizePath(dir)
	for _, part := range strings.Split(p.env["PATH"], p.pathListSeparator()) {
		if p.normalizePath(part) == normalizedDir {
			return true
		}
	}
	return false
}

func (p Paths) join(parts ...string) string {
	separator := p.separator()
	joined := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if joined == "" {
			joined = strings.TrimRight(part, separator)
			if joined == "" && part == separator {
				joined = separator
			}
			continue
		}
		part = strings.Trim(part, separator)
		if joined == separator {
			joined += part
			continue
		}
		joined += separator + part
	}
	return joined
}

func (p Paths) separator() string {
	if p.goos == "windows" {
		return `\`
	}
	return "/"
}

func (p Paths) pathListSeparator() string {
	if p.goos == "windows" {
		return ";"
	}
	return ":"
}

func (p Paths) normalizePath(value string) string {
	if p.goos == "windows" {
		return value
	}
	return path.Clean(value)
}
