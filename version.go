package fh

import (
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major:      0,
		Minor:      94,
		Patch:      0,
		PreRelease: "alpha",
		Build:      semver.Commit(),
	}
)

func Version() semver.Version {
	return version
}
