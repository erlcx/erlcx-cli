package ids

import (
	"fmt"
	"sort"
	"strings"

	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/names"
)

type outputEntry struct {
	VehiclePath string
	VehicleName string
	ImageName   string
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

		fmt.Fprintf(&b, "%s: %s\n", entry.ImageName, entry.AssetID)
	}

	return b.String(), nil
}
