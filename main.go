// Package main implements the Far Horizons CLI.
package main

import (
	"fmt"
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major:      0,
		Minor:      1,
		Patch:      0,
		PreRelease: "alpha",
		Build:      semver.Commit(),
	}
)

func main() {
	fmt.Println(version.String())
}
