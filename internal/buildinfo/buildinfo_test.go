package buildinfo

import "testing"

func TestCurrentUsesBuildVariables(t *testing.T) {
	originalVersion, originalCommit, originalBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = originalVersion, originalCommit, originalBuildTime })
	Version, Commit, BuildTime = "v1.2.3", "abc123", "2026-07-28T00:00:00Z"
	if got := Current(); got.Version != "v1.2.3" || got.Commit != "abc123" || got.BuildTime == "" {
		t.Fatalf("Current() = %+v", got)
	}
}
