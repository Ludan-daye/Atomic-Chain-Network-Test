package version

import (
	"fmt"
	"runtime"
)

// Build information. Populated via ldflags during build.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// Info represents version information
type Info struct {
	Version   string
	Commit    string
	Date      string
	BuiltBy   string
	GoVersion string
	Platform  string
}

// GetVersion returns detailed version information
func GetVersion() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns version as a formatted string
func (i Info) String() string {
	return fmt.Sprintf("NetCrate %s (%s) built on %s by %s with %s for %s",
		i.Version, i.Commit, i.Date, i.BuiltBy, i.GoVersion, i.Platform)
}

// Short returns a short version string
func (i Info) Short() string {
	if i.Version == "dev" {
		return fmt.Sprintf("NetCrate %s-%s", i.Version, i.Commit)
	}
	return fmt.Sprintf("NetCrate %s", i.Version)
}