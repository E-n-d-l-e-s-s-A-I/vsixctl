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

func formatInstallResult(r domain.ExtensionResult) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, formatError(r.Err))
	}
	return r.ID.String() + ": installed"
}

func formatUpdateResult(r domain.ExtensionResult) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, formatError(r.Err))
	}
	return r.ID.String() + ": updated"
}

func formatRemoveResult(r domain.ExtensionResult) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, formatError(r.Err))
	}
	return r.ID.String() + ": deleted"
}

func formatError(err error) string {
	for sentinel, msg := range errToMes {
		if errors.Is(err, sentinel) {
			return msg
		}
	}
	return err.Error()
}

func formatInstallPlan(requestedIDs []domain.ExtensionID, extensions []domain.DownloadInfo) string {
	extMap := make(map[domain.ExtensionID]domain.DownloadInfo, len(extensions))
	var requested, deps []domain.ExtensionID
	for _, ext := range extensions {
		extMap[ext.ID] = ext
		if slices.Contains(requestedIDs, ext.ID) {
			requested = append(requested, ext.ID)
		} else {
			deps = append(deps, ext.ID)
		}
	}
	sortIDs(requested)
	sortIDs(deps)

	var b strings.Builder
	var totalSize int64

	if len(requested) > 0 {
		fmt.Fprintf(&b, "\nExtensions (%d):\n", len(requested))
		for _, id := range requested {
			info := extMap[id]
			totalSize += info.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, info.Version, formatSize(info.Size))
		}
	}
	if len(deps) > 0 {
		fmt.Fprintf(&b, "\nDependencies (%d):\n", len(deps))
		for _, id := range deps {
			info := extMap[id]
			totalSize += info.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, info.Version, formatSize(info.Size))
		}
	}

	fmt.Fprintf(&b, "\nTotal Size: %s", formatSize(totalSize))
	return b.String()
}

func formatRemovePlan(requested []domain.ExtensionID, extensions []domain.Extension) string {
	requestedSet := make(map[domain.ExtensionID]struct{}, len(requested))
	for _, id := range requested {
		requestedSet[id] = struct{}{}
	}

	extMap := make(map[domain.ExtensionID]domain.Extension, len(extensions))
	var reqIDs, packIDs []domain.ExtensionID
	for _, ext := range extensions {
		extMap[ext.ID] = ext
		if _, ok := requestedSet[ext.ID]; ok {
			reqIDs = append(reqIDs, ext.ID)
		} else {
			packIDs = append(packIDs, ext.ID)
		}
	}
	sortIDs(reqIDs)
	sortIDs(packIDs)

	var b strings.Builder
	var totalSize int64

	if len(reqIDs) > 0 {
		fmt.Fprintf(&b, "\nExtensions (%d):\n", len(reqIDs))
		for _, id := range reqIDs {
			ext := extMap[id]
			totalSize += ext.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, ext.Version, formatSize(ext.Size))
		}
	}
	if len(packIDs) > 0 {
		fmt.Fprintf(&b, "\nPack extensions (%d):\n", len(packIDs))
		for _, id := range packIDs {
			ext := extMap[id]
			totalSize += ext.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, ext.Version, formatSize(ext.Size))
		}
	}

	fmt.Fprintf(&b, "\nTotal Size: %s", formatSize(totalSize))
	return b.String()
}

func formatUpdatePlan(toUpdate []domain.UpdateInfo) string {
	var b strings.Builder
	var totalSize int64

	sorted := slices.Clone(toUpdate)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Prev.ID.String() < sorted[j].Prev.ID.String()
	})

	if len(sorted) > 0 {
		fmt.Fprintf(&b, "\nUpdates (%d):\n", len(sorted))
		for _, u := range sorted {
			totalSize += u.New.Size
			fmt.Fprintf(&b, "  %s  %s -> %s  %s\n", u.Prev.ID, u.Prev.Version, u.New.Version, formatSize(u.New.Size))
		}
	}

	fmt.Fprintf(&b, "\nTotal Download Size: %s", formatSize(totalSize))
	return b.String()
}

func sortIDs(ids []domain.ExtensionID) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
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
