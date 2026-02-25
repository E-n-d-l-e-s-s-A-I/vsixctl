package detect

import "github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"

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
