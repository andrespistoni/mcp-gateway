//go:build windows

package proc

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	procThreadAttributeHandleList = 0x00020002
	procThreadAttributeJobList    = 0x0002000d
)

type Tree struct {
	stdin   *os.File
	stdout  *os.File
	stderr  *os.File
	process windows.Handle
	job     windows.Handle
	done    chan struct{}
	mu      sync.Mutex
	err     error
	closed  bool
}

func Start(spec ExecSpec) (*Tree, error) {
	path := spec.Executable().Path()
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo revalidar el ejecutable: %w", err)
	}
	if err := validateExecutable(path, info); err != nil {
		return nil, fmt.Errorf("el ejecutable dejó de ser válido: %w", err)
	}
	job, err := createKillOnCloseJob()
	if err != nil {
		return nil, err
	}
	childIn, parentIn, err := os.Pipe()
	if err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	parentOut, childOut, err := os.Pipe()
	if err != nil {
		closeFiles(childIn, parentIn)
		windows.CloseHandle(job)
		return nil, err
	}
	parentErr, childErr, err := os.Pipe()
	if err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut)
		windows.CloseHandle(job)
		return nil, err
	}
	children := []*os.File{childIn, childOut, childErr}
	for _, file := range children {
		if err := windows.SetHandleInformation(windows.Handle(file.Fd()), windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
			windows.CloseHandle(job)
			return nil, err
		}
	}
	attributes, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		windows.CloseHandle(job)
		return nil, err
	}
	defer attributes.Delete()
	handles := []windows.Handle{windows.Handle(childIn.Fd()), windows.Handle(childOut.Fd()), windows.Handle(childErr.Fd())}
	if err := attributes.Update(procThreadAttributeHandleList, unsafe.Pointer(&handles[0]), unsafe.Sizeof(handles[0])*uintptr(len(handles))); err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		windows.CloseHandle(job)
		return nil, err
	}
	jobs := []windows.Handle{job}
	if err := attributes.Update(procThreadAttributeJobList, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs[0])); err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		windows.CloseHandle(job)
		return nil, fmt.Errorf("el SO no admite Job Object desde creación: %w", err)
	}
	application, err := windows.UTF16PtrFromString(path)
	if err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		windows.CloseHandle(job)
		return nil, err
	}
	commandLine, err := windows.UTF16PtrFromString(buildCommandLine(path, spec.Args()))
	if err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		windows.CloseHandle(job)
		return nil, err
	}
	environment := environmentBlock(spec.Environment())
	startup := windows.StartupInfoEx{}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput = handles[0]
	startup.StdOutput = handles[1]
	startup.StdErr = handles[2]
	startup.ProcThreadAttributeList = attributes.List()
	var processInfo windows.ProcessInformation
	err = windows.CreateProcess(application, commandLine, nil, nil, true,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT,
		&environment[0], nil, &startup.StartupInfo, &processInfo)
	closeFiles(childIn, childOut, childErr)
	if err != nil {
		closeFiles(parentIn, parentOut, parentErr)
		windows.CloseHandle(job)
		return nil, err
	}
	windows.CloseHandle(processInfo.Thread)
	tree := &Tree{stdin: parentIn, stdout: parentOut, stderr: parentErr, process: processInfo.Process, job: job, done: make(chan struct{})}
	go tree.wait()
	return tree, nil
}

func createKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	information := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	information.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&information)), uint32(unsafe.Sizeof(information))); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func buildCommandLine(path string, args []string) string {
	values := make([]string, 0, len(args)+1)
	values = append(values, windows.EscapeArg(path))
	for _, argument := range args {
		values = append(values, windows.EscapeArg(argument))
	}
	return strings.Join(values, " ")
}

func environmentBlock(environment []string) []uint16 {
	joined := strings.Join(environment, "\x00") + "\x00\x00"
	return utf16.Encode([]rune(joined))
}

func closeFiles(files ...*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

func (t *Tree) Stdin() io.WriteCloser { return t.stdin }
func (t *Tree) Stdout() io.ReadCloser { return t.stdout }
func (t *Tree) Stderr() io.ReadCloser { return t.stderr }

func (t *Tree) wait() {
	_, waitErr := windows.WaitForSingleObject(t.process, windows.INFINITE)
	var exitCode uint32
	if waitErr == nil {
		waitErr = windows.GetExitCodeProcess(t.process, &exitCode)
	}
	t.mu.Lock()
	if waitErr != nil {
		t.err = waitErr
	} else if exitCode != 0 {
		t.err = fmt.Errorf("proceso terminó con código %d", exitCode)
	}
	windows.CloseHandle(t.process)
	windows.CloseHandle(t.job)
	t.closed = true
	t.mu.Unlock()
	close(t.done)
}

func (t *Tree) Wait() error {
	<-t.done
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

func (t *Tree) Terminate() error { return t.terminate(1) }
func (t *Tree) Kill() error      { return t.terminate(1) }

func (t *Tree) terminate(exitCode uint32) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	return windows.TerminateJobObject(t.job, exitCode)
}
