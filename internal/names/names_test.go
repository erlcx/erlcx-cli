package names

import "testing"

func TestFromRelativePathDerivesNames(t *testing.T) {
	got, err := FromRelativePath("Law Enforcement/Coupe - Sedan/Falcon Stallion 350 2015/Left.png")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.VehiclePath != "Law Enforcement/Coupe - Sedan/Falcon Stallion 350 2015" {
		t.Fatalf("expected vehicle path to be derived, got %q", got.VehiclePath)
	}
	if got.VehicleName != "Falcon Stallion 350 2015" {
		t.Fatalf("expected vehicle name to be derived, got %q", got.VehicleName)
	}
	if got.ImageName != "Left" {
		t.Fatalf("expected image name to be derived, got %q", got.ImageName)
	}
	if got.DisplayName != "Falcon Stallion 350 2015 - Left" {
		t.Fatalf("expected display name to be derived, got %q", got.DisplayName)
	}
}

func TestFromRelativePathRejectsImageWithoutVehicleFolder(t *testing.T) {
	_, err := FromRelativePath("Left.png")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
