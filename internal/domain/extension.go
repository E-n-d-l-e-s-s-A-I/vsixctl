package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExtensionID - уникальный идентификатор: "publisher.name"
type ExtensionID struct {
	Publisher string
	Name      string
}

func (id ExtensionID) String() string {
	return id.Publisher + "." + id.Name
}

func ParseExtensionID(s string) (ExtensionID, error) {
	splitID := strings.Split(s, ".")
	if len(splitID) != 2 {
		return ExtensionID{}, fmt.Errorf("parse extension id: invalid format %q", s)
	}
	return ExtensionID{
		Name:      splitID[1],
		Publisher: splitID[0],
	}, nil
}

// Version с семантическим версионированием
type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(s string) (Version, error) {
	splitVer := strings.Split(s, ".")
	if len(splitVer) < 2 || len(splitVer) > 3 {
		return Version{}, fmt.Errorf("parse version: invalid format %q", s)
	}
	major, err := strconv.Atoi(splitVer[0])
	if err != nil {
		return Version{}, fmt.Errorf("parse version: %w", err)
	}
	minor, err := strconv.Atoi(splitVer[1])
	if err != nil {
		return Version{}, fmt.Errorf("parse version: %w", err)
	}
	patch := 0
	if len(splitVer) == 3 {
		patch, err = strconv.Atoi(splitVer[2])
		if err != nil {
			return Version{}, fmt.Errorf("parse version: %w", err)
		}
	}

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) NewerThan(other Version) bool {
	return false
}

// Версия с источником
type VersionInfo struct {
	Version Version

	Source string
	// Запасные источники
	FallbackSources []string
}

// Extension - доменная модель расширения
type Extension struct {
	ID           ExtensionID
	Description  string
	Version      Version
	Platform     Platform      // "linux-x64", "" если универсальное
	Dependencies []ExtensionID // Для будущего дерева зависимостей
	InstalledAt  time.Time     // zero value если не установлено
}
