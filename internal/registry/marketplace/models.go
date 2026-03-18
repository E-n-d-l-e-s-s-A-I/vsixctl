package marketplace

import (
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Значения FilterType
const ExtensionIdSearch = 7
const TextSearch = 10
const DisplayNameSearch = 2

var searchTypeToFilterType = map[domain.SearchType]int{
	domain.SearchByText: TextSearch,
	domain.SearchByID:   ExtensionIdSearch,
	domain.SearchByName: DisplayNameSearch,
}

// Маски для Flags
const (
	FlagNone                = 0x0
	FlagIncludeVersions     = 0x2
	FlagIncludeFiles        = 0x4
	FlagIncludeVersionProps = 0x10
	FlagExcludeNonValidated = 0x20
	FlagIncludeAssetUri     = 0x80
	FlagIncludeStatistics   = 0x100
	FlagIncludeLatestOnly   = 0x200

	// Базовый набор флагов для запросов с версиями
	baseFlags = FlagIncludeFiles | FlagIncludeVersionProps | FlagIncludeAssetUri | FlagIncludeStatistics
)

// Значения SortBy и SortOrder
const (
	SortByRelevance = 0
	SortByInstalls  = 4
	SortByRating    = 12
	SortByName      = 2
	SortByPublished = 10
	SortByUpdated   = 1

	SortOrderAsc     = 1
	SortOrderDesc    = 2
	SortOrderDefault = 0
)

type searchCriteria struct {
	FilterType int    `json:"filterType"`
	Value      string `json:"value"`
}

type searchFilter struct {
	Criteria   []searchCriteria `json:"criteria"`
	PageNumber int              `json:"pageNumber"`
	PageSize   int              `json:"pageSize"`
	SortBy     int              `json:"sortBy"`
	SortOrder  int              `json:"sortOrder"`
}

type searchRequest struct {
	Filters    []searchFilter `json:"filters"`
	AssetTypes []string       `json:"assetTypes"`
	Flags      int            `json:"flags"`
}

type StatisticParameter struct {
	StatisticName string  `json:"statisticName"`
	Value         float32 `json:"value"`
}

type Property struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type File struct {
	AssetType string `json:"assetType"`
	Source    string `json:"source"`
}

type Version struct {
	Version          string     `json:"version"`
	Flags            string     `json:"flags"`
	LastUpdated      string     `json:"lastUpdated"`
	Files            []File     `json:"files"`
	Properties       []Property `json:"properties"`
	TargetPlatform   string     `json:"targetPlatform,omitempty"`
	AssetUri         string     `json:"assetUri"`
	FallbackAssetUri string     `json:"fallbackAssetUri"`
}

type Publisher struct {
	PublisherId   string `json:"publisherId"`
	PublisherName string `json:"publisherName"`
	DisplayName   string `json:"displayName"`
	Flags         string `json:"flags"`
}

type Extension struct {
	Publisher        Publisher            `json:"publisher"`
	ExtensionId      string               `json:"extensionId"`
	ExtensionName    string               `json:"extensionName"`
	DisplayName      string               `json:"displayName"`
	Flags            string               `json:"flags"`
	LastUpdated      time.Time            `json:"lastUpdated"`
	PublishedDate    time.Time            `json:"publishedDate"`
	ReleaseDate      time.Time            `json:"releaseDate"`
	ShortDescription string               `json:"shortDescription"`
	Versions         []Version            `json:"versions"`
	Properties       []Property           `json:"properties"`
	AssetUri         string               `json:"assetUri"`
	FallbackAssetUri string               `json:"fallbackAssetUri"`
	Categories       []string             `json:"categories"`
	Tags             []string             `json:"tags"`
	Statistics       []StatisticParameter `json:"statistics"`
}

type MetadataItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ResultMetadata struct {
	MetadataType  string         `json:"metadataType"`
	MetadataItems []MetadataItem `json:"metadataItems"`
}

type SearchResult struct {
	Extensions     []Extension      `json:"extensions"`
	ResultMetadata []ResultMetadata `json:"resultMetadata"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}
