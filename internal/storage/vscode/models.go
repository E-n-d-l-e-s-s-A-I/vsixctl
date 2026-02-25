package vscode

import "github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"

type packageJSON struct {
	Publisher   string       `json:"publisher"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Metadata    metadataJSON `json:"__metadata"`
}

type metadataJSON struct {
	// TODO: json.Unmarshal не валидирует значение — любая строка из package.json попадёт в domain.Platform
	TargetPlatform domain.Platform `json:"targetPlatform"`
}
