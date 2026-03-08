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
	domain.ErrAlreadyInstalled:      "extension already installed",
	domain.ErrVersionNotFound:       "compatible version not found",
	domain.ErrAllSourcesUnavailable: "download failed: all sources unavailable",
}

func FormatExtension(index int, ext domain.Extension) string {
	return fmt.Sprintf("%d. %s - %s", index, ext.ID, ext.Description)
}

func FormatInstallResult(r domain.InstallResult) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.ID, FormatError(r.Err))
	}
	return r.ID.String() + ": installed"
}

func FormatError(err error) string {
	for sentinel, msg := range errToMes {
		if errors.Is(err, sentinel) {
			return msg
		}
	}
	return err.Error()
}

func FormatInstallPlan(requestedIDs []domain.ExtensionID, extensions map[domain.ExtensionID]domain.VersionInfo) string {
	var requested, deps []domain.ExtensionID
	for id := range extensions {
		if slices.Contains(requestedIDs, id) {
			requested = append(requested, id)
		} else {
			deps = append(deps, id)
		}
	}
	sortIDs(requested)
	sortIDs(deps)

	var b strings.Builder
	var totalSize int64

	if len(requested) > 0 {
		fmt.Fprintf(&b, "\nExtensions (%d):\n", len(requested))
		for _, id := range requested {
			ver := extensions[id]
			totalSize += ver.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, ver.Version, FormatSize(ver.Size))
		}
	}
	if len(deps) > 0 {
		fmt.Fprintf(&b, "\nDependencies (%d):\n", len(deps))
		for _, id := range deps {
			ver := extensions[id]
			totalSize += ver.Size
			fmt.Fprintf(&b, "  %s-%s  %s\n", id, ver.Version, FormatSize(ver.Size))
		}
	}

	fmt.Fprintf(&b, "\nTotal Size: %s", FormatSize(totalSize))
	return b.String()
}

func sortIDs(ids []domain.ExtensionID) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})
}

func FormatSize(bytes int64) string {
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
