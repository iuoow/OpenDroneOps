package buildinfo

import "fmt"

// These values are replaced by the release build. Development builds retain
// safe, explicit defaults instead of claiming a version they do not have.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func Current() Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
}

func (i Info) String() string {
	return fmt.Sprintf("version=%s commit=%s build_time=%s", i.Version, i.Commit, i.BuildTime)
}
