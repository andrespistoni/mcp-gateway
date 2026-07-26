package version

import "strings"

var (
	release = "dev"
	commit  string
)

// Info contiene la identidad reproducible de la compilación.
type Info struct {
	Release string
	Commit  string
}

// Current devuelve una copia de la identidad inyectada por el linker.
func Current() Info {
	v := strings.TrimSpace(release)
	if v == "" {
		v = "dev"
	}
	return Info{Release: v, Commit: strings.TrimSpace(commit)}
}

// String produce la representación mostrada por la CLI.
func (i Info) String() string {
	if i.Commit == "" {
		return i.Release
	}
	return i.Release + " (commit " + i.Commit + ")"
}
