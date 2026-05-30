//go:build !windows

package utils

import "os"

func enableVirtualTerminalProcessing(_ *os.File) bool {
	return true
}
