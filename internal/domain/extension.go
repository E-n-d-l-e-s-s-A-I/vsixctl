package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExtensionID — уникальный идентификатор: "publisher.name"
type ExtensionID struct {
	Publisher string
	Name      string
}

type Publisher struct {
	ID   uuid.UUID
	Name string
}

func (id ExtensionID) String() string {
	return id.Publisher + "." + id.Name
}

// Version с семантическим версионированием
type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) NewerThan(other Version) bool {
	return false
}

// Extension — доменная модель расширения
type Extension struct {
	ID           uuid.UUID
	Publisher    Publisher
	Name         string
	Description  string
	Version      Version
	Dependencies []ExtensionID // Для будущего дерева зависимостей
	InstalledAt  time.Time     // zero value если не установлено
}

// SearchResult — результат поиска в маркетплейсе
type SearchResult struct {
	Extension
	DownloadCount int
	Rating        float64
}
