package vscode

import (
	"encoding/json"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type packageJSON struct {
	Publisher     string       `json:"publisher"`
	Name          string       `json:"name"`
	Version       string       `json:"version"`
	Description   string       `json:"description"`
	ExtensionPack []string     `json:"extensionPack"`
	Metadata      metadataJSON `json:"__metadata"`
}

type metadataJSON struct {
	// NOTE: json.Unmarshal не валидирует значение - любая строка из package.json попадёт в domain.Platform
	TargetPlatform domain.Platform `json:"targetPlatform"`
	Size           int64           `json:"size"`
}

// Запись в реестре расширений VS Code (extensions.json)
type registryEntry struct {
	Identifier       registryIdentifier `json:"identifier"`
	Version          string             `json:"version"`
	Location         registryLocation   `json:"location"`
	RelativeLocation string             `json:"relativeLocation"`
	Metadata         json.RawMessage    `json:"metadata,omitempty"`
}

type registryIdentifier struct {
	ID   string `json:"id"`
	UUID string `json:"uuid,omitempty"`
}

type registryLocation struct {
	Mid    int    `json:"$mid"`
	Path   string `json:"path"`
	Scheme string `json:"scheme"`
}

// registryMetadata - метаданные записи в реестре (extensions.json)
type registryMetadata struct {
	InstalledTimestamp   int64  `json:"installedTimestamp"`
	Pinned               bool   `json:"pinned"`
	Source               string `json:"source"`
	ID                   string `json:"id"`
	PublisherID          string `json:"publisherId"`
	PublisherDisplayName string `json:"publisherDisplayName"`
	TargetPlatform       string `json:"targetPlatform"`
	Updated              bool   `json:"updated"`
	IsPreReleaseVersion  bool   `json:"isPreReleaseVersion"`
	HasPreReleaseVersion bool   `json:"hasPreReleaseVersion"`
}
