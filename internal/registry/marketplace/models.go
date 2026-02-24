package marketplace

import (
	"time"

	"github.com/google/uuid"
)

// Значения FilterType
const ExtensionIdSearch = 7
const TextSearch = 10
const DisplayNameSearch = 8
const ExtensionNameSearch = 12

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
	FilterType int
	Value      string
}

type searchFilter struct {
	Criteria   []searchCriteria
	PageNumber int
	PageSize   int
	SortBy     int
	SortOrder  int
}

type searchRequest struct {
	Filters    []searchFilter
	AssetTypes []string
	Flags      int
}

type StatisticParameter struct {
	StatisticName string
	Value         float32
}

type Property struct {
	Key   string
	Value string
}

type File struct {
	AssetType string
	Source    string
}

type Version struct {
	Version     string
	Flags       string
	LastUpdated string // TODO конвертировать в datetime
	Files       []File
	Properties  []Property
}

type Publisher struct {
	PublisherId   uuid.UUID
	PublisherName string
	DisplayName   string
	Flags         string
}

type Extension struct {
	Publisher        Publisher
	ExtensionId      uuid.UUID
	ExtensionName    string
	DisplayName      string
	Flags            string
	LastUpdated      time.Time
	PublishDate      time.Time
	ReleaseDate      time.Time
	ShortDescription string
	Versions         []Version
	Properties       []Property
	AssetUri         string
	FallbackAssetUri string
	Categories       []string
	Tags             []string
	Statistics       []StatisticParameter
}

type MetadataItem struct {
	Name  string
	Count int
}

type ResultMetadata struct {
	MetadataType  string
	MetadataItems []MetadataItem
}

type SearchResult struct {
	Extensions     []Extension
	ResultMetadata []ResultMetadata
}

type SearchResponse struct {
	Results []SearchResult
}
