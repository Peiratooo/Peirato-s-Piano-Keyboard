//go:build !windows

package service

import "os"

func replaceAtomicFile(from, to string) error { return os.Rename(from, to) }
