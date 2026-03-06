package domain

import "slices"

// Platform - ос и архитектура процессора
type Platform string

const (
	LinuxX64        = Platform("linux-x64")
	LinuxArm64      = Platform("linux-arm64")
	DarwinX64       = Platform("darwin-x64")
	DarwinArm64     = Platform("darwin-arm64")
	WindowsX64      = Platform("win32-x64")
	WindowsArm64    = Platform("win32-arm64")
	UnknownPlatform = Platform("unknown")
)

var ValidPlatforms = []Platform{
	LinuxX64,
	LinuxArm64,
	DarwinX64,
	DarwinArm64,
	WindowsX64,
	WindowsArm64,
}

func IsValidPlatform(p Platform) bool {
	return slices.Contains(ValidPlatforms, p)
}
