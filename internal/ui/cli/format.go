package cli

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

var errToMes = map[error]string{
	domain.ErrNotFound:              "extension not found",
	domain.ErrNotInstalled:          "extension not installed",
	domain.ErrAlreadyInstalled:      "extension already installed",
	domain.ErrVersionNotFound:       "compatible version not found",
	domain.ErrAllSourcesUnavailable: "download failed: all sources unavailable",
}

func formatExtension(index int, ext domain.Extension) string {
	return fmt.Sprintf("%d. %s - %s", index, ext.ID, ext.Description)
}

func formatResult(r domain.ExtensionResult, successMsg string) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, formatError(r.Err))
	}
	return r.ID.String() + ": " + successMsg
}

func formatError(err error) string {
	for sentinel, msg := range errToMes {
		if errors.Is(err, sentinel) {
			return msg
		}
	}
	return err.Error()
}

// planItem - промежуточная структура для форматирования планов
type planItem struct {
	ID      domain.ExtensionID
	Version domain.Version
	Size    int64
}

// versionChangeItem - промежуточная структура для форматирования версионных изменений (reinstall, update)
type versionChangeItem struct {
	ID          domain.ExtensionID
	PrevVersion domain.Version
	NewVersion  domain.Version
	Size        int64
}

func formatInstallPlan(requestedIDs []domain.ExtensionID, extensions []domain.DownloadInfo, reinstall []domain.ReinstallInfo) string {
	items := make([]planItem, len(extensions))
	for i, ext := range extensions {
		items[i] = planItem{ID: ext.ID, Version: ext.Version, Size: ext.Size}
	}
	sections, totalSize := formatPlanSections(requestedIDs, items, "Extensions", "Dependencies")

	var b strings.Builder
	b.WriteString(sections)

	changeItems := make([]versionChangeItem, len(reinstall))
	for i, ri := range reinstall {
		changeItems[i] = versionChangeItem{
			ID:          ri.New.ID,
			PrevVersion: ri.Prev.Version,
			NewVersion:  ri.New.Version,
			Size:        ri.New.Size,
		}
	}
	reinstallSection, reinstallSize := formatVersionChangeSection("Reinstall", changeItems)
	b.WriteString(reinstallSection)
	totalSize += reinstallSize

	fmt.Fprintf(&b, "\nTotal Size: %s", formatSize(totalSize))
	return b.String()
}

func formatRemovePlan(requested []domain.ExtensionID, extensions []domain.Extension) string {
	items := make([]planItem, len(extensions))
	for i, ext := range extensions {
		items[i] = planItem{ID: ext.ID, Version: ext.Version, Size: ext.Size}
	}
	sections, totalSize := formatPlanSections(requested, items, "Extensions", "Pack extensions")

	var b strings.Builder
	b.WriteString(sections)
	fmt.Fprintf(&b, "\nTotal Size: %s", formatSize(totalSize))
	return b.String()
}

// formatPlanSections форматирует секции плана без итоговой строки.
// Возвращает отформатированные секции и суммарный размер.
func formatPlanSections(requestedIDs []domain.ExtensionID, items []planItem, requestedHeader, otherHeader string) (string, int64) {
	requestedSet := make(map[domain.ExtensionID]struct{}, len(requestedIDs))
	for _, id := range requestedIDs {
		requestedSet[id] = struct{}{}
	}

	var requested, other []planItem
	for _, item := range items {
		if _, ok := requestedSet[item.ID]; ok {
			requested = append(requested, item)
		} else {
			other = append(other, item)
		}
	}
	sortPlanItems(requested)
	sortPlanItems(other)

	var b strings.Builder
	var totalSize int64

	if len(requested) > 0 {
		fmt.Fprintf(&b, "\n%s (%d):\n", requestedHeader, len(requested))
		for _, item := range requested {
			totalSize += item.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", item.ID, item.Version, formatSize(item.Size))
		}
	}
	if len(other) > 0 {
		fmt.Fprintf(&b, "\n%s (%d):\n", otherHeader, len(other))
		for _, item := range other {
			totalSize += item.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", item.ID, item.Version, formatSize(item.Size))
		}
	}

	return b.String(), totalSize
}

func formatUpdatePlan(toUpdate []domain.UpdateInfo) string {
	items := make([]versionChangeItem, len(toUpdate))
	for i, u := range toUpdate {
		items[i] = versionChangeItem{
			ID:          u.Prev.ID,
			PrevVersion: u.Prev.Version,
			NewVersion:  u.New.Version,
			Size:        u.New.Size,
		}
	}
	section, totalSize := formatVersionChangeSection("Updates", items)

	var b strings.Builder
	b.WriteString(section)
	fmt.Fprintf(&b, "\nTotal Download Size: %s", formatSize(totalSize))
	return b.String()
}

// formatVersionChangeSection форматирует секцию с версионными изменениями (reinstall, update).
// Возвращает отформатированную секцию и суммарный размер.
func formatVersionChangeSection(header string, items []versionChangeItem) (string, int64) {
	if len(items) == 0 {
		return "", 0
	}

	sorted := slices.Clone(items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID.String() < sorted[j].ID.String()
	})

	var b strings.Builder
	var totalSize int64

	fmt.Fprintf(&b, "\n%s (%d):\n", header, len(sorted))
	for _, item := range sorted {
		totalSize += item.Size
		if item.PrevVersion == item.NewVersion {
			fmt.Fprintf(&b, "  %s-%s (reinstall)  %s\n", item.ID, item.NewVersion, formatSize(item.Size))
		} else {
			fmt.Fprintf(&b, "  %s  %s -> %s  %s\n", item.ID, item.PrevVersion, item.NewVersion, formatSize(item.Size))
		}
	}

	return b.String(), totalSize
}

func sortPlanItems(items []planItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID.String() < items[j].ID.String()
	})
}

func formatSize(bytes int64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)
	switch {
	case bytes >= gib:
		return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(gib))
	case bytes >= mib:
		return fmt.Sprintf("%.1f MiB", float64(bytes)/float64(mib))
	case bytes >= kib:
		return fmt.Sprintf("%.1f KiB", float64(bytes)/float64(kib))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
