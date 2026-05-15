package release

import "testing"

func TestArtifactName(t *testing.T) {
	cases := []struct {
		goos string
		arch string
		want string
	}{
		{"darwin", "arm64", "gzy_Darwin_arm64.tar.gz"},
		{"linux", "amd64", "gzy_Linux_x86_64.tar.gz"},
		{"windows", "amd64", "gzy_Windows_x86_64.zip"},
	}
	for _, tc := range cases {
		if got := ArtifactName(tc.goos, tc.arch); got != tc.want {
			t.Fatalf("ArtifactName(%q,%q)=%q want %q", tc.goos, tc.arch, got, tc.want)
		}
	}
}
