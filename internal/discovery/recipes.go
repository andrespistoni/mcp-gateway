package discovery

type Recipe struct {
	Name             string
	Prefix           string
	BinaryCandidates []string
	Args             []string
	InjectProject    bool
	ProjectArgument  string
}

func Recipes() []Recipe {
	return []Recipe{
		{
			Name: "codegraph", Prefix: "codegraph__",
			BinaryCandidates: []string{"~/.local/bin/codegraph", "codegraph"},
			Args:             []string{"serve", "--mcp"}, InjectProject: true, ProjectArgument: "projectPath",
		},
		{
			Name: "codebase-memory-mcp", Prefix: "cbm__",
			BinaryCandidates: []string{"~/.local/bin/codebase-memory-mcp", "codebase-memory-mcp"},
			Args:             []string{}, InjectProject: true, ProjectArgument: "projectPath",
		},
		{
			Name: "engram", Prefix: "engram__",
			BinaryCandidates: []string{"~/.local/bin/engram", "engram"},
			Args:             []string{"mcp", "--tools=agent"}, InjectProject: true, ProjectArgument: "projectPath",
		},
	}
}
