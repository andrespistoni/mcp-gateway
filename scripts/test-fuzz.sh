#!/usr/bin/env bash
set -euo pipefail

duration="${1:-3s}"

go test ./internal/config -run='^$' -fuzz=FuzzDecode -fuzztime="$duration"
go test ./internal/mcp -run='^$' -fuzz=FuzzEnvelope -fuzztime="$duration"
go test ./internal/mcp -run='^$' -fuzz=FuzzCodec -fuzztime="$duration"
go test ./internal/claude -run='^$' -fuzz=FuzzProjectMerge -fuzztime="$duration"
go test ./internal/sse -run='^$' -fuzz=FuzzGateParser -fuzztime="$duration"
go test ./internal/sse -run='^$' -fuzz=FuzzGateDifferential -fuzztime="$duration"
go test ./internal/sse -run='^$' -fuzz=FuzzSessionID -fuzztime="$duration"
go test ./internal/sse -run='^$' -fuzz=FuzzCursor -fuzztime="$duration"
