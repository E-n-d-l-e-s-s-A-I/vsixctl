package domain

// SearchResult - результат поиска в маркетплейсе
type SearchResult struct {
	Extension
	DownloadCount int
	Rating        float64
}

// InstallResult - результат установки расширения
type InstallResult struct {
	ID  ExtensionID
	Err error
}
