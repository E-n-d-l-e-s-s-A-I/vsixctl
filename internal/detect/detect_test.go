package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name         string
		os           string
		arch         string
		wantPlatform domain.Platform
	}{
		{
			name:         "linux_amd64",
			os:           "linux",
			arch:         "amd64",
			wantPlatform: domain.LinuxX64,
		},
		{
			name:         "linux_arm64",
			os:           "linux",
			arch:         "arm64",
			wantPlatform: domain.LinuxArm64,
		},
		{
			name:         "darwin_amd64",
			os:           "darwin",
			arch:         "amd64",
			wantPlatform: domain.DarwinX64,
		},
		{
			name:         "darwin_arm64",
			os:           "darwin",
			arch:         "arm64",
			wantPlatform: domain.DarwinArm64,
		},
		{
			name:         "win_amd64",
			os:           "windows",
			arch:         "amd64",
			wantPlatform: domain.WindowsX64,
		},
		{
			name:         "win_arm64",
			os:           "windows",
			arch:         "arm64",
			wantPlatform: domain.WindowsArm64,
		},
		{
			name:         "unknown_os",
			os:           "temple_os",
			arch:         "amd64",
			wantPlatform: domain.UnknownPlatform,
		},
		{
			name:         "unknown_arch",
			os:           "windows",
			arch:         "amd32",
			wantPlatform: domain.UnknownPlatform,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			detectedPlatform := DetectPlatform(testCase.os, testCase.arch)
			if testCase.wantPlatform != detectedPlatform {
				t.Errorf("got %+v, want %+v", detectedPlatform, testCase.wantPlatform)
			}
		})
	}
}

func TestDetectExtensionsDir(t *testing.T) {
	tests := []struct {
		name                string
		vscodeExtensionsEnv string
		setupDirs           []string // директории которые нужно создать внутри homeDir
		wantRelPath         string   // ожидаемый путь относительно homeDir ("" если wantFullPath задан)
		wantFullPath        string   // ожидаемый полный путь (для env-кейса)
	}{
		{
			name:                "env_var_set",
			vscodeExtensionsEnv: "/custom/extensions",
			wantFullPath:        "/custom/extensions",
		},
		{
			name:        "vscode_exists",
			setupDirs:   []string{".vscode/extensions"},
			wantRelPath: filepath.Join(".vscode", "extensions"),
		},
		{
			name:        "insiders_only",
			setupDirs:   []string{".vscode-insiders/extensions"},
			wantRelPath: filepath.Join(".vscode-insiders", "extensions"),
		},
		{
			name:        "both_exist_prefers_stable",
			setupDirs:   []string{".vscode/extensions", ".vscode-insiders/extensions"},
			wantRelPath: filepath.Join(".vscode", "extensions"),
		},
		{
			name:        "neither_exists_defaults_stable",
			wantRelPath: filepath.Join(".vscode", "extensions"),
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			homeDir := t.TempDir()

			for _, dir := range testCase.setupDirs {
				os.MkdirAll(filepath.Join(homeDir, dir), 0o755)
			}

			got := DetectExtensionsDir(homeDir, testCase.vscodeExtensionsEnv)

			expected := testCase.wantFullPath
			if expected == "" {
				expected = filepath.Join(homeDir, testCase.wantRelPath)
			}

			if got != expected {
				t.Errorf("got %s, want %s", got, expected)
			}
		})
	}
}
