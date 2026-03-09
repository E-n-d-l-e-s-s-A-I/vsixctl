package domain

// SearchResult - результат поиска в маркетплейсе
type SearchResult struct {
	Extension
	DownloadCount int
	Rating        float64
}

// ExtensionResult - результат установки расширения
type ExtensionResult struct {
	ID  ExtensionID
	Err error
}
