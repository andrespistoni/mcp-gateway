//go:build windows

package persist

import (
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const replaceFileWriteThrough = 0x00000001

var replaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func replacePath(source, destination string) error {
	sourcePtr, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPtr, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		result, _, callErr := replaceFileW.Call(
			uintptr(unsafe.Pointer(destinationPtr)),
			uintptr(unsafe.Pointer(sourcePtr)),
			0,
			replaceFileWriteThrough,
			0,
			0,
		)
		if result == 0 {
			if callErr != nil && !errors.Is(callErr, syscall.Errno(0)) {
				return callErr
			}
			return syscall.EINVAL
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return windows.MoveFileEx(sourcePtr, destinationPtr, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func syncDirectory(string) error {
	return nil
}
