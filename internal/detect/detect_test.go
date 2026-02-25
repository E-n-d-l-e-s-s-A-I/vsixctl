package detect

import (
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
