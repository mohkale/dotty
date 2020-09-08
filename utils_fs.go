package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/rs/zerolog/log"
)

func _pathExists(path string, extraCheck func(os.FileInfo) bool, def bool) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		errno := err.(*os.PathError).Err.(syscall.Errno)
		if errno == syscall.ENOTDIR {
			return false, nil
		}
		return def, err
	}
	return extraCheck(stat), nil
}

func pathExists(path string) (bool, error) {
	return _pathExists(path, func(_ os.FileInfo) bool { return true }, false)
}

func fileExists(path string) (bool, error) {
	return _pathExists(path, func(fi os.FileInfo) bool { return !fi.IsDir() }, false)
}

func dirExists(path string) (bool, error) {
	return _pathExists(path, func(fi os.FileInfo) bool { return fi.IsDir() }, false)
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
