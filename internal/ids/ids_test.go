package ids

import (
	"strings"
	"testing"
	"time"

	"github.com/erlcx/cli/internal/lockfile"
)

const validSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestGenerateWritesDeterministicIDsText(t *testing.T) {
	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "123456"})
	lock.Files["Law Enforcement/Coupe - Sedan/Falcon Stallion 350 2015/Left.png"] = entry("300")
	lock.Files["Law Enforcement/Coupe - Sedan/Falcon Stallion 350 2015/Back.png"] = entry("100")
	lock.Files["Law Enforcement/SUV/Bullhorn BH15 SSV 2009/Right1.png"] = entry("500")
	lock.Files["Law Enforcement/SUV/Bullhorn BH15 SSV 2009/Left1.png"] = entry("400")
	lock.Files["Civilian/Falcon Stallion 350 2015/Top.png"] = entry("200")

	got, err := Generate(lock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := strings.Join([]string{
		"Falcon Stallion 350 2015",
		"Top: 200",
		"",
		"Falcon Stallion 350 2015",
		"Back: 100",
		"Left: 300",
		"",
		"Bullhorn BH15 SSV 2009",
		"Left1: 400",
		"Right1: 500",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("expected IDs text:\n%q\ngot:\n%q", want, got)
	}
}

func TestGenerateRejectsInvalidLockFile(t *testing.T) {
	_, err := Generate(lockfile.LockFile{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateRejectsEntryWithoutVehicleFolder(t *testing.T) {
	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "123456"})
	lock.Files["Left.png"] = entry("100")

	_, err := Generate(lock)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func entry(assetID string) lockfile.Entry {
	return lockfile.Entry{
		SHA256:      validSHA256,
		AssetType:   lockfile.AssetTypeDecal,
		AssetID:     assetID,
		DisplayName: "display",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}
}
