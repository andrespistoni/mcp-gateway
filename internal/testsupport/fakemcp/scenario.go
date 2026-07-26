package fakemcp

type Scenario string

const (
	Healthy            Scenario = "healthy"
	Paginated          Scenario = "paginated"
	CursorCycle        Scenario = "cursor-cycle"
	InvalidTools       Scenario = "invalid-tools"
	MissingTools       Scenario = "missing-tools"
	InvalidTool        Scenario = "invalid-tool"
	InvalidCursor      Scenario = "invalid-cursor"
	HundredPages       Scenario = "hundred-pages"
	TooManyPages       Scenario = "too-many-pages"
	MaxTools           Scenario = "max-tools"
	TooManyTools       Scenario = "too-many-tools"
	Delayed            Scenario = "delayed"
	Batch              Scenario = "batch"
	EmptyLine          Scenario = "empty-line"
	PartialEOF         Scenario = "partial-eof"
	Stderr             Scenario = "stderr"
	ProcessTree        Scenario = "process-tree"
	RuntimeHealthy     Scenario = "runtime-healthy"
	RuntimePaginated   Scenario = "runtime-paginated"
	RuntimeCRLF        Scenario = "runtime-crlf"
	RuntimeInvalid     Scenario = "runtime-invalid"
	RuntimeLargeStderr Scenario = "runtime-large-stderr"
	CollisionShort     Scenario = "collision-short"
	CollisionLong      Scenario = "collision-long"
)
