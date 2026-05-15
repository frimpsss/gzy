package paths

import "testing"

func TestConfigFileDarwin(t *testing.T) {
	paths := New("darwin", map[string]string{"HOME": "/Users/alex"})

	got := paths.ConfigFile()
	want := "/Users/alex/Library/Application Support/gzy/config.json"
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigFileLinux(t *testing.T) {
	paths := New("linux", map[string]string{"HOME": "/home/alex"})

	got := paths.ConfigFile()
	want := "/home/alex/.config/gzy/config.json"
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigFileLinuxRespectsXDGConfigHome(t *testing.T) {
	paths := New("linux", map[string]string{
		"HOME":            "/home/alex",
		"XDG_CONFIG_HOME": "/tmp/xdg",
	})

	got := paths.ConfigFile()
	want := "/tmp/xdg/gzy/config.json"
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestConfigFileWindowsUsesAppData(t *testing.T) {
	paths := New("windows", map[string]string{
		"APPDATA":     `C:\Users\alex\AppData\Roaming`,
		"USERPROFILE": `C:\Users\alex`,
	})

	got := paths.ConfigFile()
	want := `C:\Users\alex\AppData\Roaming\gzy\config.json`
	if got != want {
		t.Fatalf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestDefaultKeyPair(t *testing.T) {
	paths := New("linux", map[string]string{"HOME": "/home/alex"})

	privateKey, publicKey := paths.DefaultKeyPair("work")
	if want := "/home/alex/.ssh/gzy_work"; privateKey != want {
		t.Fatalf("private key = %q, want %q", privateKey, want)
	}
	if want := "/home/alex/.ssh/gzy_work.pub"; publicKey != want {
		t.Fatalf("public key = %q, want %q", publicKey, want)
	}
}

func TestBinDirLinuxPrefersLocalBinWhenInPath(t *testing.T) {
	paths := New("linux", map[string]string{
		"HOME": "/home/alex",
		"PATH": "/usr/bin:/home/alex/.local/bin:/bin",
	})

	got := paths.BinDir()
	want := "/home/alex/.local/bin"
	if got != want {
		t.Fatalf("BinDir() = %q, want %q", got, want)
	}
}

func TestBinDirLinuxFallsBackToHomeBin(t *testing.T) {
	paths := New("linux", map[string]string{
		"HOME": "/home/alex",
		"PATH": "/usr/bin:/bin",
	})

	got := paths.BinDir()
	want := "/home/alex/bin"
	if got != want {
		t.Fatalf("BinDir() = %q, want %q", got, want)
	}
}

func TestBinDirWindowsUsesUserProfile(t *testing.T) {
	paths := New("windows", map[string]string{"USERPROFILE": `C:\Users\alex`})

	got := paths.BinDir()
	want := `C:\Users\alex\bin`
	if got != want {
		t.Fatalf("BinDir() = %q, want %q", got, want)
	}
}
