package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/testsupport/fakemcp"
)

type isolatedEnvironment struct {
	root    string
	home    string
	bin     string
	project string
	gateway string
	fakeMCP string
	env     []string
}

func newIsolatedEnvironment(t *testing.T) *isolatedEnvironment {
	t.Helper()
	root := repositoryRoot(t)
	directory := t.TempDir()
	bin := filepath.Join(directory, "bin")
	for _, path := range []string{bin, filepath.Join(directory, "home"), filepath.Join(directory, "project")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	environment := &isolatedEnvironment{
		root:    root,
		home:    filepath.Join(directory, "home"),
		bin:     bin,
		project: filepath.Join(directory, "project"),
		gateway: filepath.Join(bin, executableName("mcp-gateway")),
		fakeMCP: filepath.Join(bin, executableName("fake-mcp")),
	}
	buildGoBinary(t, root, environment.gateway, "./cmd/mcp-gateway")
	if err := fakemcp.Build(context.Background(), environment.fakeMCP); err != nil {
		t.Fatalf("compilar fake MCP: %v", err)
	}
	buildGoBinary(t, root, filepath.Join(bin, executableName(daemonCommand())), "./test/e2e/testdata/fake-systemctl")
	environment.env = append(os.Environ(),
		"HOME="+environment.home,
		"XDG_CONFIG_HOME="+filepath.Join(environment.home, ".config"),
		"PATH="+environment.bin,
	)
	return environment
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(workingDirectory, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func daemonCommand() string {
	switch runtime.GOOS {
	case "darwin":
		return "launchctl"
	case "windows":
		return "schtasks"
	default:
		return "systemctl"
	}
}

func buildGoBinary(t *testing.T, root, output, packagePath string) {
	t.Helper()
	command := exec.Command("go", "build", "-o", output, packagePath)
	command.Dir = root
	outputBytes, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compilar %s: %v\n%s", packagePath, err, outputBytes)
	}
}

func (e *isolatedEnvironment) run(t *testing.T, arguments ...string) string {
	t.Helper()
	command := exec.Command(e.gateway, arguments...)
	command.Dir = e.project
	command.Env = e.env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", e.gateway, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func (e *isolatedEnvironment) writeRuntimeConfiguration(t *testing.T, port int) {
	t.Helper()
	configurationDirectory := filepath.Join(e.home, ".mcp-gateway")
	if err := os.MkdirAll(configurationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	configuration := fmt.Sprintf(`version: 1
port: %d
downstreams:
  - name: alpha
    prefix: alpha__
    binary: %s
    args: ["--scenario", "runtime-healthy"]
    enabled: true
    env: {}
  - name: beta
    prefix: beta__
    binary: %s
    args: ["--scenario", "runtime-healthy"]
    enabled: true
    env: {}
`, port, strconv.Quote(e.fakeMCP), strconv.Quote(e.fakeMCP))
	if err := os.WriteFile(filepath.Join(configurationDirectory, "mcp-downstreams.yaml"), []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
}

func dynamicPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func (e *isolatedEnvironment) startServe(t *testing.T, port int) *exec.Cmd {
	t.Helper()
	command := exec.Command(e.gateway, "serve", "--port", strconv.Itoa(port))
	command.Dir = e.project
	command.Env = e.env
	command.Stdout = &bytes.Buffer{}
	command.Stderr = &bytes.Buffer{}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopServe(t, command) })
	return command
}

func stopServe(t *testing.T, command *exec.Cmd) {
	t.Helper()
	if command.Process == nil {
		return
	}
	_ = command.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		_ = command.Process.Kill()
		<-done
		t.Error("serve no terminó tras la señal de limpieza")
	}
}

func openSSE(t *testing.T, port int) (*http.Response, *bufio.Reader) {
	t.Helper()
	url := "http://localhost:" + strconv.Itoa(port) + "/sse"
	deadline := time.Now().Add(5 * time.Second)
	for {
		response, err := (&http.Client{}).Get(url)
		if err == nil && response.StatusCode == http.StatusOK {
			return response, bufio.NewReader(response.Body)
		}
		if response != nil {
			_ = response.Body.Close()
		}
		if time.Now().After(deadline) {
			t.Fatalf("abrir SSE en puerto dinámico %d: %v", port, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func readSSEEvent(t *testing.T, reader *bufio.Reader, expected string) []byte {
	t.Helper()
	event, err := reader.ReadString('\n')
	if err != nil || event != "event: "+expected+"\n" {
		t.Fatalf("evento SSE = %q, %v; se esperaba %q", event, err, expected)
	}
	data, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(data, "data: ") {
		t.Fatalf("data SSE = %q, %v", data, err)
	}
	separator, err := reader.ReadString('\n')
	if err != nil || separator != "\n" {
		t.Fatalf("separador SSE = %q, %v", separator, err)
	}
	return []byte(strings.TrimSuffix(strings.TrimPrefix(data, "data: "), "\n"))
}

func postJSONRPC(t *testing.T, port int, endpointPath, payload string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "http://localhost:"+strconv.Itoa(port)+endpointPath, strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusAccepted || len(body) != 0 {
		t.Fatalf("POST %s = %d, body=%q", payload, response.StatusCode, body)
	}
}

func requireResultID(t *testing.T, data []byte, id int) map[string]json.RawMessage {
	t.Helper()
	var message map[string]json.RawMessage
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if string(message["id"]) != strconv.Itoa(id) || message["result"] == nil {
		t.Fatalf("respuesta JSON-RPC = %s", data)
	}
	return message
}
