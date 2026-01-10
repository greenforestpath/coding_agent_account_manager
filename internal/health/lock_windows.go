//go:build windows

package health

import (
	"os"
)

// No-op for Windows to allow compilation without complex syscall logic
func lockFile(f *os.File) error {
	return nil
}

func unlockFile(f *os.File) error {
	return nil
}
