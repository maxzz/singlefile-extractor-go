//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	winEnableProcessedOutput             = 0x0001
	winEnableWrapAtEOLOutput             = 0x0002
	winEnableVirtualTerminalProcessing   = 0x0004
)

var (
	winKernel32          = syscall.NewLazyDLL("kernel32.dll")
	winGetConsoleModeProc = winKernel32.NewProc("GetConsoleMode")
	winSetConsoleModeProc = winKernel32.NewProc("SetConsoleMode")
)

func enableVirtualTerminalProcessing(f *os.File) bool {
	if f == nil {
		return false
	}

	h := syscall.Handle(f.Fd())
	var mode uint32
	r1, _, _ := winGetConsoleModeProc.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return false
	}

	newMode := mode | winEnableProcessedOutput | winEnableWrapAtEOLOutput | winEnableVirtualTerminalProcessing
	r1, _, _ = winSetConsoleModeProc.Call(uintptr(h), uintptr(newMode))
	return r1 != 0
}

