package fh

import (
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major:      0,
		Minor:      92,
		Patch:      0,
		PreRelease: "alpha",
		Build:      semver.Commit(),
	}
)

func Version() semver.Version {
	return version
}
