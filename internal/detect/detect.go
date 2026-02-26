package detect

import (
	"os"
	"path/filepath"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

var platformMap = map[string]domain.Platform{
	"linux_amd64":   domain.LinuxX64,
	"linux_arm64":   domain.LinuxArm64,
	"darwin_amd64":  domain.DarwinX64,
	"darwin_arm64":  domain.DarwinArm64,
	"windows_amd64": domain.WindowsX64,
	"windows_arm64": domain.WindowsArm64,
}

func DetectPlatform(goos, goarch string) domain.Platform {
	platform, ok := platformMap[goos+"_"+goarch]
	if !ok {
		platform = domain.UnknownPlatform
	}
	return platform
}

func DetectExtensionsDir(homeDir string, vscodeExtensionsEnv string) string {
	if vscodeExtensionsEnv != "" {
		return vscodeExtensionsEnv
	}
	candidates := []string{
		filepath.Join(homeDir, ".vscode", "extensions"),
		filepath.Join(homeDir, ".vscode-insiders", "extensions"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}
