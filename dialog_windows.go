package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// showMessage triggers a native Windows dialog box.
func showMessage(title, msg string, isError bool) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")

	var uType uint = 0x00000000 // MB_OK
	if isError {
		uType |= 0x00000010 // MB_ICONERROR
	} else {
		uType |= 0x00000040 // MB_ICONINFORMATION
	}

	tPtr, _ := syscall.UTF16PtrFromString(title)
	mPtr, _ := syscall.UTF16PtrFromString(msg)

	messageBox.Call(0, uintptr(unsafe.Pointer(mPtr)), uintptr(unsafe.Pointer(tPtr)), uintptr(uType))
}

// logToDisk writes messages to a log file in the AppData/Local/Temp folder for debugging.
func logToDisk(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)

	// Create or append to a log file in Temp directory
	logPath := filepath.Join(os.TempDir(), "AssistantUpdater.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		// Add a newline if it doesn't have one
		if !strings.HasSuffix(formatted, "\n") {
			formatted += "\n"
		}
		f.WriteString(formatted)
	}
}
