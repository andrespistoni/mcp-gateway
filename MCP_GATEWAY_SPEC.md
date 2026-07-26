# mcp-gateway - Product and Technical Specification

## 1. Document status

- Status: Ready for SDD implementation
- Project name: `mcp-gateway`
- Repository name: `mcp-gateway`
- Go binary and CLI command: `mcp-gateway`
- Claude Code MCP entry name: `mcp-gateway`
- Default daemon/service name: `mcp-gateway`
- Initial transport: MCP over HTTP/SSE
- Default endpoint: `http://localhost:3333/sse`

This document is the source specification for the SDD implementation flow of this repository.

## 2. Summary

`mcp-gateway` is a local Go application that exposes multiple locally installed MCP servers through one HTTP/SSE endpoint on `localhost`.

Claude Code connects only to the gateway:

```text
Claude Code
    |
    | HTTP/SSE: http://localhost:3333/sse?projectDir=<absolute-path>
    v
mcp-gateway
    |-- MCP server A over stdio
    |-- MCP server B over stdio
    `-- MCP server C over stdio
```

The application must not require compile-time knowledge of every MCP server. It supports:

1. Discovery of a small catalog of common MCP servers in known locations and in `PATH`.
2. User configuration for arbitrary MCP stdio servers.
3. One user-level daemon and downstream registry.
4. One project-level Claude Code registration carrying the project directory.

The gateway implements a generic HTTP/SSE-to-stdio MCP bridge. All required behavior is defined by this specification.

## 3. Problem statement

The corporate Claude Code configuration blocks normal local MCP registration and permits MCP access only through HTTP/SSE URLs whose hostname is exactly `localhost`.

The team needs to use locally installed MCP servers without configuring each one directly in Claude Code. Each developer may have different installation paths, environment variables and optional MCP servers.

The solution must provide one local HTTP/SSE gateway that:

- Is accepted by the corporate allowlist.
- Starts and communicates with local MCP servers over stdio.
- Aggregates their tools under collision-safe names.
- Can discover selected common MCP installations.
- Can be configured for unknown MCP servers.
- Preserves the project directory for project-aware tools.
- Is simple to install and operate through a CLI.

## 4. Goals

- Expose local stdio MCP servers through `http://localhost:<port>/sse`.
- Use the literal hostname `localhost` in every generated URL.
- Run as one user-level background service.
- Keep downstream MCP configuration at user scope.
- Register the gateway independently in each project.
- Discover known MCP servers from default paths and `PATH`.
- Validate discovered MCP servers with a real MCP handshake.
- Allow arbitrary downstream MCP servers to be added manually.
- Merge tools from all active downstreams.
- Route tool calls by a configurable tool-name prefix.
- Support Linux, macOS and Windows user services.
- Provide deterministic commands suitable for scripts and team documentation.

## 5. Non-goals for the first release

- A terminal UI or graphical UI.
- Binding to an IP address or a non-local interface.
- Exposing the gateway to the local network.
- Discovering every possible MCP server without a recipe or configuration.
- Installing downstream MCP packages automatically.
- Proxying remote HTTP or SSE downstream MCP servers.
- Aggregating MCP resources, prompts, sampling or elicitation.
- Hot-reloading configuration while the daemon is running.
- Sharing secrets or absolute installation paths in the project repository.
- Replacing the MCP configuration facilities of other AI clients.

## 6. Fixed decisions

### 6.1 Host and transport

- The first release uses MCP over HTTP/SSE with one persistent SSE response stream and a session-specific HTTP message endpoint.
- The server listens using `localhost:<port>`.
- Generated Claude Code URLs use `http://localhost:<port>/sse`.
- The application must never substitute `127.0.0.1`, `::1`, `0.0.0.0`, a machine hostname or another IP address.
- The hostname is not configurable in the first release.
- The default port is `3333` and may be changed with `--port`.
- Allowed ports are `1024` through `65535`.
- Port precedence is CLI `--port`, then the user configuration, then `3333`.

### 6.2 Configuration scopes

User scope owns:

- The gateway daemon.
- The listening port.
- The downstream MCP definitions.
- MCP executable paths, arguments and environment variable references.
- Discovery results.

Project scope owns:

- The `.mcp.json` entry used by Claude Code.
- The absolute project directory carried in the SSE URL.

There is no project-level downstream configuration in the first release. A future release may add project-specific enable/disable overlays if a concrete need appears.

### 6.3 User interface

- The first release is CLI-only.
- A TUI is explicitly deferred because discovery, configuration and diagnostics can be handled with small deterministic commands.

### 6.4 Downstream transport

- The first release supports stdio downstream MCP servers only.
- Downstream commands are executed directly with `exec.CommandContext`.
- Commands must never be executed through a shell.

## 7. User journeys

### 7.1 First-time setup

```bash
mcp-gateway setup
```

Expected behavior:

1. Create the user configuration directory when absent.
2. Discover known MCP servers.
3. Write valid discoveries to the user configuration without deleting existing entries.
4. Install or update the user daemon.
5. Start or restart the daemon.
6. Print the discovered MCP servers and the next project-registration command.

### 7.2 Register a project

```bash
cd /path/to/project
mcp-gateway register-project
```

Expected behavior:

1. Resolve the current directory to an absolute path.
2. Validate that the directory exists.
3. Merge the `mcp-gateway` entry into `.mcp.json`.
4. Preserve unrelated MCP entries.
5. Use a URL containing `localhost` and the URL-encoded project directory.
6. Add `.mcp.json` to `.gitignore` if not already present.
7. Report whether the entry was created or updated.

### 7.3 Add an unknown MCP server

```bash
mcp-gateway add custom \
  --prefix custom__ \
  --binary /opt/company/bin/custom-mcp \
  --arg serve
```

Expected behavior:

1. Validate the name, prefix, binary and arguments.
2. Resolve the binary path.
3. Perform `initialize` and `tools/list` against the candidate.
4. Save the entry only if validation succeeds.
5. Restart the daemon when it is managed and running.

### 7.4 Add a project-aware MCP server

```bash
mcp-gateway add source-index \
  --prefix source__ \
  --binary source-index-mcp \
  --inject-project projectPath
```

The gateway injects the current session's project directory into the `projectPath` tool argument only when the caller did not already provide that argument.

### 7.5 Diagnose an installation

```bash
mcp-gateway doctor
```

The command checks configuration parsing, binary resolution, duplicate names and prefixes, downstream MCP handshakes, port availability or daemon status, the SSE endpoint and Claude Code availability.

## 8. CLI specification

### 8.1 Commands

```text
mcp-gateway setup [--port 3333]
mcp-gateway discover [--write]
mcp-gateway add <name> --prefix <prefix__> --binary <path-or-name> [options]
mcp-gateway remove <name>
mcp-gateway enable <name>
mcp-gateway disable <name>
mcp-gateway list
mcp-gateway doctor [--verbose]
mcp-gateway serve [--port 3333]
mcp-gateway enable-daemon [--port 3333]
mcp-gateway disable-daemon
mcp-gateway restart
mcp-gateway register-project [--project-dir <path>] [--port 3333]
mcp-gateway install-claude [--port 3333]
mcp-gateway version
mcp-gateway help
```

### 8.2 `discover`

- Without `--write`, discovery is read-only.
- With `--write`, valid discoveries are merged into the user configuration.
- Existing entries are never overwritten automatically.
- Missing candidates are reported as skipped, not as errors.
- Exit code is `0` when discovery completes, even if no known MCP is installed.
- Invalid discovered candidates are reported with their handshake error.

### 8.3 `add`

Supported options:

```text
--prefix <value>                 Required; must end in "__"
--binary <path-or-name>          Required
--arg <value>                    Repeatable
--env <KEY=VALUE>                Repeatable
--inject-project <argument-name> Optional
--disabled                       Save without enabling
--skip-validation                Optional escape hatch for temporarily unavailable MCPs
```

`--skip-validation` must produce a warning and is not used by discovery.

### 8.4 `list`

Example output:

```text
NAME                     STATUS       PREFIX       BINARY
codegraph                available    codegraph__  /home/user/.local/bin/codegraph
codebase-memory-mcp      available    cbm__        /home/user/.local/bin/codebase-memory-mcp
custom                   unavailable  custom__     /opt/company/bin/custom-mcp
```

### 8.5 `install-claude`

This optional command registers a user-scoped Claude Code connection:

```text
http://localhost:3333/sse
```

It is intended only for project-independent use. `register-project` remains the recommended command because it carries `projectDir`.

The command must:

- Verify that the `claude` CLI exists in `PATH`.
- Invoke `claude mcp add` using SSE transport and user scope.
- Treat an already-existing matching registration as success.
- Never generate an IP-based URL.

### 8.6 Exit codes

- `0`: operation completed successfully.
- `1`: validation, configuration, process, network or registration error.
- `2`: invalid CLI usage or arguments.

Errors go to stderr. Normal and machine-readable status output goes to stdout.

## 9. User configuration

### 9.1 Location

The canonical configuration is:

```text
~/.mcp-gateway/mcp-downstreams.yaml
```

The directory is created with user-only permissions where supported. The configuration file is written with mode `0600` on Unix-like systems because it may contain environment references or values.

### 9.2 Schema

```yaml
version: 1

port: 3333

downstreams:
  - name: codegraph
    prefix: "codegraph__"
    binary: "~/.local/bin/codegraph"
    args: ["serve", "--mcp"]
    enabled: true
    env: {}
    inject_project: true
    project_argument: "projectPath"

  - name: custom
    prefix: "custom__"
    binary: "/opt/company/bin/custom-mcp"
    args: ["serve"]
    enabled: true
    env:
      API_TOKEN: "${CUSTOM_MCP_API_TOKEN}"
    inject_project: false
```

### 9.3 Field rules

- `version` is required and must be `1`.
- `port` is optional and defaults to `3333`.
- `name` is required and unique.
- `prefix` is required, unique and must end with `__`.
- `binary` is required.
- `args` defaults to an empty list.
- `enabled` defaults to `true`.
- `env` defaults to an empty map.
- `inject_project` defaults to `false`.
- `project_argument` defaults to `projectPath` when injection is enabled.
- Unknown fields produce a configuration error instead of being silently ignored.
- `~` is expanded only at the start of filesystem paths.
- A non-existing configured path falls back to `exec.LookPath(filepath.Base(binary))`.
- `${NAME}` environment references are expanded from the daemon environment.
- A missing referenced environment variable makes that downstream unavailable and produces a diagnostic without stopping other downstreams.

### 9.4 Persistence behavior

- User edits are authoritative.
- Setup and discovery merge but never overwrite an existing downstream with the same name.
- Writes are atomic using a temporary file and rename.
- Invalid configuration must not be partially written.
- Configuration changes take effect after daemon restart.

## 10. Discovery specification

### 10.1 Discovery model

Discovery uses embedded recipes. A recipe contains:

```go
type DiscoveryRecipe struct {
    Name             string
    Prefix           string
    BinaryCandidates []string
    Args             []string
    InjectProject    bool
    ProjectArgument  string
}
```

Discovery is intentionally limited to known recipes because arbitrary binaries cannot be safely or reliably identified as MCP servers.

### 10.2 Initial recipes

```yaml
- name: codegraph
  prefix: "codegraph__"
  binary_candidates:
    - "~/.local/bin/codegraph"
    - "codegraph"
  args: ["serve", "--mcp"]
  inject_project: true
  project_argument: "projectPath"

- name: codebase-memory-mcp
  prefix: "cbm__"
  binary_candidates:
    - "~/.local/bin/codebase-memory-mcp"
    - "codebase-memory-mcp"
  args: []
  inject_project: true
  project_argument: "projectPath"

- name: engram
  prefix: "engram__"
  binary_candidates:
    - "~/.local/bin/engram"
    - "engram"
  args: ["mcp", "--tools=agent"]
  inject_project: true
  project_argument: "projectPath"
```

Recipe arguments must be verified against the versions distributed in the target corporate environment before release. Updating embedded recipes does not change the generic proxy implementation.

### 10.3 Search order

For each recipe:

1. Expand and check explicit filesystem candidates in recipe order.
2. Use `exec.LookPath` for name-only candidates.
3. Deduplicate paths after resolving symlinks where supported.
4. Validate the first candidate that completes the MCP handshake.
5. Report all failed candidate validations in verbose mode.

### 10.4 Handshake validation

Discovery starts the candidate with its recipe arguments and performs:

1. JSON-RPC `initialize` using MCP protocol version `2024-11-05`.
2. `notifications/initialized`.
3. `tools/list`.
4. Verification that the response is valid JSON-RPC and contains a tools array.
5. Graceful process termination.

The complete validation has a configurable internal timeout with a default of five seconds. Timeout is reported as an unavailable candidate, not a fatal discovery error.

### 10.5 Future importers

Importing existing `.mcp.json`, Claude Desktop, VS Code, Cursor or OpenCode configurations is deferred. It may be added later as explicit `import` commands. The first release must not scan or modify third-party configuration files beyond its own project registration.

## 11. Claude project registration

### 11.1 Generated entry

`register-project` merges this structure into `<projectDir>/.mcp.json`:

```json
{
  "mcpServers": {
    "mcp-gateway": {
      "type": "sse",
      "url": "http://localhost:3333/sse?projectDir=<url-encoded-absolute-path>"
    }
  }
}
```

### 11.2 Merge rules

- Preserve every unrelated top-level key.
- Preserve every unrelated `mcpServers` entry.
- Create or replace only `mcpServers.mcp-gateway`.
- Produce stable, indented JSON followed by a newline.
- Return an error for malformed existing JSON instead of overwriting it.
- Write atomically.
- Add `.mcp.json` to the project `.gitignore` exactly once.
- Do not modify a global Git ignore file.

### 11.3 Project directory rules

- Default to the current working directory.
- Accept an override through `--project-dir`.
- Resolve to an absolute, cleaned path.
- Require that the path exists and is a directory.
- URL-encode the query parameter.
- The SSE server validates the path again when the session is opened.

## 12. HTTP/SSE server behavior

### 12.1 Endpoints

```text
GET  /sse
POST /message?sessionId=<id>
```

No other public endpoint is required in the first release.

### 12.2 SSE session creation

`GET /sse` must:

1. Require HTTP streaming support.
2. Resolve the optional `projectDir` from the query parameter.
3. Optionally accept `X-Project-Dir` as a compatibility fallback.
4. When a project directory is present, validate that it exists.
5. Generate a cryptographically random session ID.
6. Store the project directory, including an empty value, for that session.
7. Return `Content-Type: text/event-stream`.
8. Disable caching and proxy buffering.
9. Send an `endpoint` event containing `/message?sessionId=<id>`.
10. Keep the connection open until client disconnect or server shutdown.

### 12.3 Message handling

`POST /message` must:

- Reject methods other than POST.
- Require a valid active `sessionId`.
- Decode one MCP JSON-RPC request.
- Route the request.
- Publish the JSON-RPC response to the matching SSE session.
- Return HTTP `202 Accepted` after delivery.
- Return bounded timeout and shutdown errors instead of blocking indefinitely.

### 12.4 MCP methods exposed to Claude

The first release supports:

- `initialize`
- `notifications/initialized`
- `ping`
- `tools/list`
- `tools/call`

Unsupported methods return JSON-RPC method-not-found.

### 12.5 Shutdown

- Handle SIGINT and SIGTERM.
- Stop accepting new HTTP requests.
- Allow active HTTP requests up to five seconds to finish.
- Close all SSE sessions.
- Terminate all downstream processes.
- Wait for downstream worker goroutines.

## 13. Downstream process management

### 13.1 Startup

At gateway startup:

1. Load and validate user configuration.
2. Resolve enabled downstream binaries.
3. Skip unavailable downstreams with a warning.
4. Start each available process using stdio pipes.
5. Send `initialize`.
6. Send `notifications/initialized`.
7. Request `tools/list`.
8. Prefix and cache each tool definition.
9. Start a serialized worker for tool calls.

One broken downstream must not prevent the gateway from serving healthy downstreams.

### 13.2 Tool naming

Given downstream prefix `codegraph__` and tool `explore`, Claude sees:

```text
codegraph__explore
```

Rules:

- Prefixes must be unique.
- Prefixed tool names must be globally unique.
- Startup fails on a tool-name collision.
- `tools/list` returns only tools from downstreams that initialized successfully.

### 13.3 Tool routing

For `tools/call`:

1. Parse the prefixed tool name.
2. Identify the downstream by prefix.
3. Remove the prefix.
4. Optionally inject the session project directory.
5. Forward the original tool name and arguments over stdio.
6. Return the downstream result without changing its content.

### 13.4 Project injection

When `inject_project` is true:

- Arguments must be a JSON object or empty.
- Add `project_argument: projectDir` only when the session has a non-empty project directory and the key is absent.
- Never overwrite a value supplied by Claude.
- If arguments are not a JSON object, forward them unchanged.

### 13.5 Process failure

- EOF, malformed protocol output or process exit marks the downstream unavailable.
- Calls to an unavailable downstream return an MCP tool error.
- Automatic process restart is deferred; `mcp-gateway restart` restores the downstream.
- Downstream stderr may be captured in bounded diagnostic logs but must never be mixed with stdout JSON-RPC traffic.

## 14. Daemon behavior

### 14.1 Service command

The service runs:

```bash
mcp-gateway serve --port 3333
```

### 14.2 Platforms

- Linux: systemd user service.
- macOS: launchd LaunchAgent.
- Windows: user scheduled task.

### 14.3 Requirements

- `enable-daemon` is idempotent.
- Re-running it updates the binary path and port.
- If an old daemon is running, it is stopped before replacement.
- `restart` reports a clear error when no supported service manager exists.
- Service files and task names use `mcp-gateway`.
- Service definitions must not contain an IP-based endpoint.

## 15. Security requirements

- Listen only on `localhost`.
- Do not offer a flag to bind another host in the first release.
- Execute downstreams directly, never through a shell.
- Discovery may execute only embedded known recipes.
- Arbitrary commands are executed only after explicit `add` or manual user configuration.
- Do not log environment values or tool arguments by default.
- Redact values of keys containing `TOKEN`, `SECRET`, `PASSWORD`, `KEY` or `AUTH` in verbose diagnostics.
- Limit JSON-RPC input size to 1 MiB by default.
- Bound response delivery and handshake timeouts.
- Generate session IDs using `crypto/rand`.
- Validate `projectDir` existence before storing it.
- Do not follow or modify third-party MCP configurations automatically.
- Local authentication is not required in the first release because corporate access is restricted to the local `localhost` endpoint; adding authentication is a future option.

## 16. Architecture

### 16.1 Suggested package structure

```text
cmd/mcp-gateway/             CLI entry point
internal/cli/                Command parsing and output
internal/config/             YAML model, load, validate and atomic save
internal/discovery/          Embedded recipes and candidate validation
internal/mcp/                JSON-RPC and MCP protocol types
internal/proxy/              Downstream processes and routing
internal/sse/                HTTP/SSE sessions and handlers
internal/claude/             .mcp.json and optional Claude CLI registration
internal/daemon/             systemd, launchd and Windows task integration
internal/version/            Build version
```

### 16.2 Dependency direction

```text
CLI -> application services -> config/proxy/SSE/daemon adapters
```

Protocol and configuration validation should remain independent from filesystem and process execution through small interfaces so tests can use fakes.

### 16.3 Dependencies

Prefer the standard library. The expected external dependency is:

```text
gopkg.in/yaml.v3
```

A dedicated MCP Go SDK is not required for the first release because the required feature set is a small JSON-RPC and MCP tools subset. Adopting one should require a concrete simplification and compatibility validation.

## 17. Component responsibilities

The implementation is divided into the following responsibilities:

| Component | Responsibility |
|---|---|
| HTTP/SSE server | Listener lifecycle, sessions, `/sse`, `/message` and graceful shutdown |
| MCP protocol | JSON-RPC request, response, error, initialization and tool types |
| Downstream proxy | stdio process lifecycle, handshake, tool discovery and call routing |
| Configuration | YAML parsing, strict validation, environment expansion and atomic persistence |
| Discovery | Embedded recipes, path search and handshake validation |
| Project registrar | `.mcp.json` merge, URL generation and `.gitignore` update |
| Claude registrar | Optional user-scope registration through the Claude CLI |
| Daemon manager | systemd, launchd and Windows scheduled-task integration |
| CLI | Argument parsing, command orchestration, stdout/stderr and exit codes |

## 18. Observability and diagnostics

Normal daemon logs should include:

- Listening address as `localhost:<port>`.
- Config file location.
- Downstream start success or failure.
- Number of tools loaded per downstream.
- Session open and close at debug level only.
- Graceful shutdown progress.

Logs must not include full tool arguments, environment values or secrets.

`doctor --verbose` may show commands and arguments but must redact sensitive environment values.

## 19. Testing strategy

### 19.1 Unit tests

- Configuration validation and unknown-field rejection.
- `~` and environment expansion.
- Binary resolution and `PATH` fallback.
- Discovery candidate ordering and deduplication.
- Discovery merge without overwrite.
- Prefix validation and collision detection.
- Project argument injection without overwrite.
- `.mcp.json` merge and malformed-file handling.
- `.gitignore` idempotency.
- URL generation always uses `localhost`.
- Port range validation.
- Platform-specific service file generation.

### 19.2 Integration tests

Use a fake stdio MCP executable that can:

- Complete initialization.
- Return a configurable tool list.
- Echo tool calls.
- Return protocol errors.
- Emit malformed output.
- Exit unexpectedly.
- Delay responses to exercise timeouts.

Verify:

- Gateway startup with zero downstreams.
- Gateway startup with healthy and unhealthy downstreams.
- Aggregated `tools/list`.
- Correct prefixed routing.
- Per-session project injection.
- Concurrent SSE sessions for different projects.
- Graceful shutdown.

### 19.3 End-to-end tests

- Run `setup` in an isolated home directory.
- Run `discover --write` with fake binaries in known paths and `PATH`.
- Run `register-project` and inspect `.mcp.json`.
- Connect an SSE test client to `http://localhost:<test-port>/sse`.
- Perform initialize, tools/list and tools/call.
- Confirm no generated file contains `127.0.0.1`, `::1` or `0.0.0.0`.

Tests must use dynamic available ports except tests specifically validating the default `3333` configuration.

## 20. Acceptance criteria

The first release is accepted when all of the following are true:

1. `mcp-gateway serve` listens on `localhost:3333` by default.
2. Claude Code connects using an SSE URL whose host is exactly `localhost`.
3. `register-project` preserves existing `.mcp.json` entries.
4. Separate projects create separate SSE sessions with their own project directories.
5. The gateway starts configured stdio MCP servers and completes MCP initialization.
6. `tools/list` returns the union of healthy downstream tools with prefixes.
7. `tools/call` reaches the correct downstream with its original unprefixed name.
8. Project injection is configurable and never overwrites a caller-provided value.
9. Failure of one downstream does not make healthy downstreams unavailable.
10. `discover` finds valid initial recipes in default paths and `PATH`.
11. Discovery rejects binaries that fail MCP initialization or `tools/list`.
12. Arbitrary MCP servers can be added without changing or recompiling the application.
13. The daemon can be installed for Linux, macOS and Windows.
14. `doctor` identifies missing binaries, invalid config, port problems and handshake failures.
15. No generated URL or service configuration uses an IP address.
16. Unit, integration and end-to-end tests pass.
17. The produced binary is standalone and contains only gateway, discovery, configuration and service-management functionality.

## 21. Suggested implementation slices

### Slice 1: Protocol and configuration

- Create Go module and CLI shell.
- Define protocol types.
- Implement YAML load, validation and atomic save.
- Implement binary and environment resolution.

### Slice 2: Downstream proxy

- Spawn stdio processes.
- Implement initialize and tools/list.
- Prefix tools.
- Route tools/call.
- Implement project argument injection.

### Slice 3: HTTP/SSE gateway

- Implement `/sse` and `/message`.
- Implement session-bound project directories.
- Aggregate tool lists and calls.
- Implement graceful shutdown.

### Slice 4: Discovery and management CLI

- Embed initial recipes.
- Implement `discover`, `add`, `remove`, `enable`, `disable`, `list` and `doctor`.
- Implement `setup` orchestration.

### Slice 5: Claude and daemon integration

- Implement `register-project`.
- Implement optional user-scope `install-claude`.
- Port systemd, launchd and Windows scheduled-task support.
- Implement restart and disable commands.

### Slice 6: Hardening and release

- Complete timeout, size-limit and redaction behavior.
- Add integration and end-to-end tests.
- Cross-compile release binaries.
- Write installation and team onboarding documentation.

## 22. Definition of done

- All acceptance criteria are demonstrated by automated tests or documented manual verification where OS service managers cannot be exercised in CI.
- `go test ./... -count=1` passes.
- `go vet ./...` passes.
- Release binaries are produced for the supported team platforms.
- A fresh user can install, discover MCPs, start the daemon and register a project using only the README.
- A user can add an unknown stdio MCP without recompiling the gateway.
- Claude Code successfully invokes at least two different downstream MCP servers through one `http://localhost:<port>/sse` connection.

## 23. Future options

These are intentionally not part of the first release:

- Importers for existing MCP client configuration files.
- Project-specific downstream enable/disable overlays.
- Streamable HTTP transport in addition to SSE, if allowed by corporate policy.
- HTTP/SSE downstream MCP support.
- Dynamic configuration reload.
- Automatic downstream restart with backoff.
- Resources and prompts aggregation.
- A TUI built over the stable CLI/application services.
- A signed corporate discovery-recipe catalog.

## 24. Blocking open questions

There are no blocking product or architecture questions for starting the SDD flow.

Before producing the first distributable release, the implementation team must verify only environment-specific facts:

- Exact supported operating systems in the team.
- Exact installed command arguments for the initial CodeGraph, codebase-memory-mcp and Engram versions.
- Whether port `3333` is available and approved on all managed machines.
- Whether corporate policy requires an additional approval prompt before starting configured local binaries.

These checks do not change the architecture or prevent implementation from starting.
