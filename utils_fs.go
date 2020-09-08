package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/rs/zerolog/log"
)

func _pathExists(path string, extraCheck func(os.FileInfo) bool) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			errno := err.(*os.PathError).Err.(syscall.Errno)
			if errno == syscall.ENOTDIR {
				return false, nil
			}
			return false, err
		}
	}
	return extraCheck(stat), nil
}

func pathExists(path string) (bool, error) {
	return _pathExists(path, func(_ os.FileInfo) bool { return true })
}

func fileExists(path string) (bool, error) {
	return _pathExists(path, func(fi os.FileInfo) bool { return !fi.IsDir() })
}

func dirExists(path string) (bool, error) {
	return _pathExists(path, func(fi os.FileInfo) bool { return fi.IsDir() })
}

/**
 * return the first path in paths that points to an existing file.
 * if there's an error while stating any file, immeadiately returns
 * cancelling any pending file checks.
 */
func findExistingFile(paths ...string) (string, error) {
	for _, path := range paths {
		log.Trace().Str("path", path).Msg("Checking for file at path")
		exists, err := fileExists(path)
		if err != nil {
			return "", err
		}
		if exists {
			return path, nil
		}
	}

	return "", fmt.Errorf("Unable to find any existing file")
}
