package pkg

import (
	"fmt"
	"os"
	"syscall"

	"github.com/rs/zerolog/log"
)

func PathExistsCheck(path string, extraCheck func(os.FileInfo) bool, def bool, followSymlinks bool) (bool, error) {
	var stat os.FileInfo
	var err error
	if followSymlinks {
		stat, err = os.Stat(path)
	} else {
		stat, err = os.Lstat(path)
	}

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

func pathExists(path string, followSymlinks bool) (bool, error) {
	return PathExistsCheck(path, func(_ os.FileInfo) bool { return true }, false, followSymlinks)
}

func fileExists(path string, followSymlinks bool) (bool, error) {
	return PathExistsCheck(path, func(fi os.FileInfo) bool { return !fi.IsDir() }, false, followSymlinks)
}

func dirExists(path string, followSymlinks bool) (bool, error) {
	return PathExistsCheck(path, func(fi os.FileInfo) bool { return fi.IsDir() }, false, followSymlinks)
}

/**
 * return the first path in paths that points to an existing file.
 * if there's an error while stating any file, immeadiately returns
 * cancelling any pending file checks.
 */
func FindExistingFile(paths ...string) (string, error) {
	for _, path := range paths {
		log.Trace().Str("path", path).Msg("Checking for file at path")
		exists, err := fileExists(path, true)
		if err != nil {
			return "", err
		}
		if exists {
			return path, nil
		}
	}

	return "", fmt.Errorf("Unable to find any existing file")
}
