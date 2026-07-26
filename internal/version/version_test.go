package version

import "testing"

func TestCurrentDeterministicDev(t *testing.T) {
	originalRelease, originalCommit := release, commit
	t.Cleanup(func() { release, commit = originalRelease, originalCommit })
	release, commit = "", ""

	if got := Current().String(); got != "dev" {
		t.Fatalf("Current().String() = %q, se esperaba dev", got)
	}
}

func TestCurrentInjectedReleaseAndCommit(t *testing.T) {
	originalRelease, originalCommit := release, commit
	t.Cleanup(func() { release, commit = originalRelease, originalCommit })
	release, commit = "1.2.3", "abc123"

	if got := Current().String(); got != "1.2.3 (commit abc123)" {
		t.Fatalf("Current().String() = %q", got)
	}
}
