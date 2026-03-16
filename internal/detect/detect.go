package detect

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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

// DetectPlatform определяет платформу VS Code по runtime.GOOS и runtime.GOARCH.
func DetectPlatform(goos, goarch string) domain.Platform {
	platform, ok := platformMap[goos+"_"+goarch]
	if !ok {
		platform = domain.UnknownPlatform
	}
	return platform
}

// DetectExtensionsDir определяет путь к директории расширений VS Code.
// Приоритет:
//  1. $VSCODE_EXTENSIONS
//  2. ~/.vscode/extensions
//  3. ~/.vscode-insiders/extensions
//  4. ~/.vscode-server/extensions
//
// Если ни одна директория не найдена, возвращает стандартный путь ~/.vscode/extensions.
func DetectExtensionsDir(homeDir string, vscodeExtensionsEnv string) string {
	if vscodeExtensionsEnv != "" {
		return vscodeExtensionsEnv
	}
	candidates := []string{
		filepath.Join(homeDir, ".vscode", "extensions"),
		filepath.Join(homeDir, ".vscode-insiders", "extensions"),
		filepath.Join(homeDir, ".vscode-server", "extensions"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

// DetectVscodeVer определяет версию vscode
func DetectVscodeVer(ctx context.Context) (domain.Version, error) {
	out, err := exec.CommandContext(ctx, "code", "--version").Output()
	if err != nil {
		return domain.Version{}, fmt.Errorf("detect vscode version: %w", err)
	}
	line, _, _ := bytes.Cut(out, []byte("\n"))
	ver, err := domain.ParseVersion(string(line))
	if err != nil {
		return domain.Version{}, fmt.Errorf("detect vscode version: %w", err)
	}
	return ver, nil
}
