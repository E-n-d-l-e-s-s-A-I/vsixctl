package domain

import "fmt"

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

func (r InstallResult) String() string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, r.Err)
	}
	return r.ID.String() + ": installed"
}
