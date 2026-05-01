package ids

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/names"
)

type outputEntry struct {
	VehiclePath string
	VehicleName string
	ImageName   string
	FileName    string
	RelPath     string
	AssetID     string
}

func Generate(lock lockfile.LockFile) (string, error) {
	if err := lockfile.Validate(lock); err != nil {
		return "", err
	}

	entries := make([]outputEntry, 0, len(lock.Files))
	for relPath, entry := range lock.Files {
		derived, err := names.FromRelativePath(relPath)
		if err != nil {
			return "", err
		}
		entries = append(entries, outputEntry{
			VehiclePath: derived.VehiclePath,
			VehicleName: derived.VehicleName,
			ImageName:   derived.ImageName,
			FileName:    path.Base(relPath),
			RelPath:     relPath,
			AssetID:     entry.AssetID,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].VehiclePath != entries[j].VehiclePath {
			return strings.ToLower(entries[i].VehiclePath) < strings.ToLower(entries[j].VehiclePath)
		}
		if entries[i].ImageName != entries[j].ImageName {
			return strings.ToLower(entries[i].ImageName) < strings.ToLower(entries[j].ImageName)
		}
		return strings.ToLower(entries[i].RelPath) < strings.ToLower(entries[j].RelPath)
	})

	duplicateNames := duplicateImageNamesByVehicle(entries)

	var b strings.Builder
	currentVehiclePath := ""
	for i, entry := range entries {
		if entry.VehiclePath != currentVehiclePath {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(entry.VehicleName)
			b.WriteByte('\n')
			currentVehiclePath = entry.VehiclePath
		}

		fmt.Fprintf(&b, "%s: %s\n", entry.OutputName(duplicateNames), entry.AssetID)
	}

	return b.String(), nil
}

func duplicateImageNamesByVehicle(entries []outputEntry) map[string]bool {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[duplicateKey(entry)]++
	}

	duplicates := map[string]bool{}
	for key, count := range counts {
		if count > 1 {
			duplicates[key] = true
		}
	}
	return duplicates
}

func duplicateKey(entry outputEntry) string {
	return strings.ToLower(entry.VehiclePath) + "\x00" + strings.ToLower(entry.ImageName)
}

func (entry outputEntry) OutputName(duplicateNames map[string]bool) string {
	if duplicateNames[duplicateKey(entry)] {
		return entry.FileName
	}
	return entry.ImageName
}
