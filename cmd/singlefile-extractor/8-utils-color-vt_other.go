//go:build !windows

package main

import "os"

func enableVirtualTerminalProcessing(_ *os.File) bool {
	return true
}

