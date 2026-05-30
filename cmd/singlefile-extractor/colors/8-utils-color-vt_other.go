//go:build !windows

package colors

import "os"

func enableVirtualTerminalProcessing(_ *os.File) bool {
	return true
}
