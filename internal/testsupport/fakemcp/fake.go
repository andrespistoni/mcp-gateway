package fakemcp

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"mcp-gateway/internal/mcp"
)

func Build(ctx context.Context, destination string) error {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("no se pudo localizar fake-mcp")
	}
	command := exec.CommandContext(ctx, "go", "build", "-o", destination, ".")
	command.Dir = filepath.Join(filepath.Dir(source), "cmd", "fake-mcp")
	command.Stdin = nil
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("no se pudo compilar fake-mcp: %w (%s)", err, output)
	}
	return nil
}

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("fake-mcp", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	scenarioValue := set.String("scenario", string(Healthy), "escenario")
	descendant := set.String("descendant", "", "proceso descendiente")
	marker := set.String("marker", "", "archivo de PIDs")
	events := set.String("events", "", "archivo de eventos")
	if err := set.Parse(args); err != nil {
		return 2
	}
	if *descendant != "" {
		return runDescendant(*descendant, *marker)
	}
	scenario := Scenario(*scenarioValue)
	if scenario == ProcessTree {
		if err := startChild(*marker); err != nil {
			return 3
		}
		if *marker != "" && !waitForDescendants(*marker) {
			return 3
		}
	}
	protocolOutput := stdout
	if scenario == RuntimeCRLF {
		protocolOutput = crlfWriter{writer: stdout}
	}
	codec := mcp.NewCodec(stdin, protocolOutput)
	initialize, err := codec.Read()
	if err != nil || initialize.Kind() != mcp.Request || initialize.Method() != "initialize" {
		return 4
	}
	initializeID, _ := initialize.ID()
	appendEvent(*events, "initialize")
	response, _ := mcp.NewResult(initializeID, map[string]any{
		"protocolVersion": mcp.ProtocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]any{"name": "fake-mcp", "version": "test"},
		"future":          map[string]any{"preserved": true},
	})
	if err := codec.Write(response); err != nil {
		return 5
	}
	initialized, err := codec.Read()
	if err != nil || initialized.Kind() != mcp.Notification || initialized.Method() != "notifications/initialized" {
		return 6
	}
	if scenario == Stderr {
		_, _ = io.WriteString(stderr, "AUTH_TOKEN=valor-que-no-debe-exponerse\n")
	}
	if scenario == RuntimeLargeStderr {
		_, _ = stderr.Write(bytes.Repeat([]byte("stderr-runtime\n"), 8192))
	}
	if scenario == Batch {
		_, _ = io.WriteString(stdout, "[]\n")
		return 0
	}
	if scenario == EmptyLine {
		_, _ = io.WriteString(stdout, "\n")
		return 0
	}
	if scenario == PartialEOF {
		_, _ = io.WriteString(stdout, `{"jsonrpc":"2.0"`)
		return 0
	}
	if scenario == Delayed {
		time.Sleep(30 * time.Second)
		return 0
	}
	for {
		request, err := codec.Read()
		if err != nil {
			return 0
		}
		if request.Kind() != mcp.Request || request.Method() != "tools/list" {
			return 7
		}
		id, _ := request.ID()
		appendEvent(*events, "tools/list")
		result := toolsPage(scenario, request.Params())
		response, _ := mcp.NewResult(id, result)
		if err := codec.Write(response); err != nil {
			return 8
		}
		if !continuesAfterPage(scenario, request.Params()) {
			if persistentScenario(scenario) {
				if scenario == RuntimeInvalid {
					_, _ = io.WriteString(stdout, "invalid runtime output\n")
				}
				return serveRuntimeCalls(codec)
			}
			return 0
		}
	}
}

func serveRuntimeCalls(codec *mcp.Codec) int {
	for {
		request, err := codec.Read()
		if err != nil {
			return 0
		}
		if request.Kind() != mcp.Request || request.Method() != "tools/call" {
			return 7
		}
		id, _ := request.ID()
		result, _ := mcp.NewResult(id, map[string]any{"echo": json.RawMessage(request.Params()), "futureResult": true})
		if err := codec.Write(result); err != nil {
			return 8
		}
	}
}

func toolsPage(scenario Scenario, params json.RawMessage) any {
	switch scenario {
	case Paginated, RuntimePaginated:
		if pageCursor(params) == "next" {
			return map[string]any{"tools": []any{tool("second", 2)}}
		}
		return map[string]any{"tools": []any{tool("first", 1)}, "nextCursor": "next"}
	case CursorCycle:
		return map[string]any{"tools": []any{tool("cycle", 1)}, "nextCursor": "same"}
	case InvalidTools:
		return map[string]any{"tools": "invalid"}
	case MissingTools:
		return map[string]any{"other": []any{}}
	case InvalidTool:
		return map[string]any{"tools": []any{map[string]any{"description": "missing name"}}}
	case InvalidCursor:
		return map[string]any{"tools": []any{}, "nextCursor": 7}
	case HundredPages, TooManyPages:
		index := pageIndex(params)
		result := map[string]any{"tools": []any{tool(fmt.Sprintf("tool-%d", index), index)}}
		if scenario == TooManyPages || index < 99 {
			result["nextCursor"] = fmt.Sprintf("p%d", index+1)
		}
		return result
	case MaxTools, TooManyTools:
		count := 5000
		if scenario == TooManyTools {
			count++
		}
		tools := make([]any, count)
		for i := range tools {
			tools[i] = tool(fmt.Sprintf("tool-%d", i), i)
		}
		return map[string]any{"tools": tools}
	case CollisionShort:
		return map[string]any{"tools": []any{tool("b__echo", 1)}}
	case CollisionLong:
		return map[string]any{"tools": []any{tool("echo", 1)}}
	default:
		return map[string]any{"tools": []any{tool("echo", 1)}, "futureResult": true}
	}
}

func tool(name string, order int) map[string]any {
	return map[string]any{
		"name":        name,
		"description": "fake",
		"inputSchema": map[string]any{"type": "object"},
		"future":      map[string]any{"order": order},
	}
}

func pageCursor(params json.RawMessage) string {
	fields, err := mcp.DecodeObject(params)
	if err != nil {
		return ""
	}
	var cursor string
	_ = json.Unmarshal(fields["cursor"], &cursor)
	return cursor
}

func pageIndex(params json.RawMessage) int {
	cursor := pageCursor(params)
	if cursor == "" {
		return 0
	}
	index, _ := strconv.Atoi(strings.TrimPrefix(cursor, "p"))
	return index
}

func continuesAfterPage(scenario Scenario, params json.RawMessage) bool {
	switch scenario {
	case Paginated, RuntimePaginated:
		return pageCursor(params) == ""
	case CursorCycle, TooManyPages:
		return true
	case HundredPages:
		return pageIndex(params) < 99
	default:
		return false
	}
}

func persistentScenario(scenario Scenario) bool {
	switch scenario {
	case RuntimeHealthy, RuntimePaginated, RuntimeCRLF, RuntimeInvalid, RuntimeLargeStderr, CollisionShort, CollisionLong:
		return true
	default:
		return false
	}
}

type crlfWriter struct {
	writer io.Writer
}

func (w crlfWriter) Write(data []byte) (int, error) {
	replaced := bytes.ReplaceAll(data, []byte{'\n'}, []byte{'\r', '\n'})
	if _, err := w.writer.Write(replaced); err != nil {
		return 0, err
	}
	return len(data), nil
}

func appendEvent(path, event string) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(file, event)
	_ = file.Close()
}

func startChild(marker string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable, "--descendant=child", "--marker="+marker)
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Start()
}

func runDescendant(level, marker string) int {
	if marker != "" {
		file, err := os.OpenFile(marker, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(file, "%s:%d\n", level, os.Getpid())
			_ = file.Close()
		}
	}
	if level == "child" {
		executable, err := os.Executable()
		if err != nil {
			return 9
		}
		command := exec.Command(executable, "--descendant=grandchild", "--marker="+marker)
		command.Stdin = nil
		command.Stdout = io.Discard
		command.Stderr = io.Discard
		if err := command.Start(); err != nil {
			return 10
		}
	}
	select {}
}

func waitForDescendants(marker string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(marker)
		if err == nil && bytes.Count(data, []byte{'\n'}) >= 2 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
