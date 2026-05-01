package names

import (
	"fmt"
	"path"

	"github.com/erlcx/cli/internal/scanner"
)

type Derived struct {
	VehiclePath string
	VehicleName string
	ImageName   string
	DisplayName string
}

func FromRelativePath(relPath string) (Derived, error) {
	relPath = scanner.CleanRelativePath(relPath)
	if relPath == "" {
		return Derived{}, fmt.Errorf("derive names: relative path must not be empty")
	}

	vehiclePath := path.Dir(relPath)
	if vehiclePath == "." || vehiclePath == "" {
		return Derived{}, fmt.Errorf("derive names for %s: image must be inside a vehicle folder", relPath)
	}

	fileName := path.Base(relPath)
	imageName := fileName[:len(fileName)-len(path.Ext(fileName))]
	if imageName == "" {
		return Derived{}, fmt.Errorf("derive names for %s: image name must not be empty", relPath)
	}

	vehicleName := path.Base(vehiclePath)
	return Derived{
		VehiclePath: vehiclePath,
		VehicleName: vehicleName,
		ImageName:   imageName,
		DisplayName: fmt.Sprintf("%s - %s", vehicleName, imageName),
	}, nil
}
