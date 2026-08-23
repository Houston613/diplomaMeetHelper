package version

import (
	"fmt"
)

var (
	Version = "v0.1.0-dev"
)

// PrintBuildInfo outputs the build metadata to stdout.
func PrintBuildInfo() {
	fmt.Printf("DiplomaMeetHelper %s)\n", Version)
}
