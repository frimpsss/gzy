package release

func ArtifactName(goos string, arch string) string {
	osName := map[string]string{"darwin": "Darwin", "linux": "Linux", "windows": "Windows"}[goos]
	archName := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[arch]
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "gzy_" + osName + "_" + archName + ext
}
