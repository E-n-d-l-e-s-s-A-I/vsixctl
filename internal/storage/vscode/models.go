package vscode

type packageJSON struct {
	Publisher   string       `json:"publisher"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Metadata    metadataJSON `json:"__metadata"`
}

type metadataJSON struct {
	TargetPlatform string `json:"targetPlatform"`
}
