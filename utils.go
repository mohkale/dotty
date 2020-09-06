package main

import (
	"fmt"
	"os"
	fp "path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
)

type recursiveBuildPathErrorCallback = func(base string, arg interface{})

/**
 * Recursively build a path from a slice of strings of arbitrary depth.
 *
 * The only arguments this function can handle are strings, and
 * slices of strings. If at some depth it encounters a type other
 * than this, it passes the path built so far and the current argument
 * to `err` and skips to the next path.
 *
 * How a path is constructed should be intuitive.
 *  ("foo" "bar" "baz") ;; is passed to ch as is.
 *  (("foo" "bar" "baz")) ;; is the same as above
 *
 * Wrapping in parens indicates a hierarchy for paths:
 *  (("foo" ("bar")))) ;; foo/bar
 *
 * The top level of the slice doesn't contribute to a hierarchy.
 *  ("foo" ("bar") "baz") ;; foo bar baz
 *
 * Multiple files at a lower depth lead to newer paths in a higher depth.
 * This happens in a predictable and consistent way:
 *  (("foo" ("bar" "baz" ("bag")))) ;; foo/bar/bag foo/baz/bag
 *
 * Nesting doesn't forward track, meaning deeper directories only affect
 * directories higher up in their parent depth. In this case a path with
 * "bag" isn't prefixed to "bar" or "baz". "bag" is passed as is.
 *  (("foo" ("bar" "baz") "bag")) ;; foo/bar foo/baz bag
 *
 * On the other hand, if you nest once more, you'll backtrack from the first
 * path, ignoring any other nested paths on the way.
 *  (("foo" ("bar") "baz" ("bag"))) ;; foo/bar foo/bag baz/bag
 *
 */
func recursiveBuildPath(
	ch chan string,
	paths AnySlice,
	base string,
	preJoin func(string) (string, bool),
	err recursiveBuildPathErrorCallback,
) {
	var recursiveDo func(paths AnySlice, base string)
	recursiveDo = func(paths AnySlice, base string) {
		lastRecurse := 0
		i, currentPaths := 0, make([]string, len(paths))

		for _, path := range paths {
			if path == nil {
				path = ""
			}

			if pathStr, ok := path.(string); ok {
				if pathStr, ok := preJoin(pathStr); ok {
					currentPaths[i] = joinPath(base, pathStr)
					i++
				}
			} else if pathSlice, ok := path.(AnySlice); ok {
				for j, dir := range currentPaths {
					if j < i {
						recursiveDo(pathSlice, dir)
					}
				}
				lastRecurse = i
			} else {
				err(base, path)
			}
		}
		for j := lastRecurse; j < i; j++ {
			ch <- currentPaths[j]
		}
	}

	for _, path := range paths {
		if pathStr, ok := path.(string); ok {
			if pathStr, ok := preJoin(pathStr); ok {
				ch <- joinPath(base, pathStr)
			}
		} else if pathSlice, ok := path.(AnySlice); ok {
			recursiveDo(pathSlice, base)
		} else {
			err(base, path)
		}
	}

	close(ch)
}

/**
 * An abstraction over recursiveBuildPath designed to build directives
 * from sequences of paths.
 *
 * This function automates a lot of the boilerplate involved in managing
 * context and options while recursively building paths. It firstly builds
 * all paths using recursiveBuildPath. Every complete path is passed back
 * to pathCompleteCallback. When it encounters a map (indicating a structure
 * that modifies options before recursing further into other paths). It clones
 * the current context and calls updateContext to extract and apply any options.
 * It then recurses into the result of getSrcsFromOpts which simply repeats this
 * process at a deeper level.
 *
 * TODO add test coverage
 */
func recursiveBuildDirectivesFromPaths(
	ctx *Context, args AnySlice,
	pathCompleteCallback func(ctx *Context, p string),
	getSrcsFromOpts func(opts map[Any]Any) (Any, bool),
	updateContext func(ctx *Context, opts map[Any]Any) *Context,
) {
	pathCh, done := make(chan string), make(chan struct{})
	go func() {
		for path := range pathCh {
			pathCompleteCallback(ctx, path)
		}
		done <- struct{}{}
	}()

	go recursiveBuildPath(pathCh, args, ctx.cwd, ctx.eval, func(base string, arg Any) {
		// the only situation in which the input can not be a path is when
		// it's a map containing perhaps more directories.
		sMap, ok := arg.(map[Any]Any)
		if !ok {
			log.Warn().Str("directive", fmt.Sprintf("%v", arg)).
				Msgf("directive must be a map of symbols to options, not %T", arg)
			return
		}

		if src, ok := getSrcsFromOpts(sMap); ok {
			newCtx := updateContext(ctx.chdir(base), sMap)
			if srcStr, ok := src.(string); ok {
				src = AnySlice{srcStr}
			}

			if srcSlice, ok := src.(AnySlice); ok {
				recursiveBuildDirectivesFromPaths(newCtx, srcSlice,
					pathCompleteCallback, getSrcsFromOpts, updateContext)
			} else {
				log.Warn().
					Str("path", fmt.Sprintf("%s", src)).
					Msgf("path must be a string or list of strings, not %T", src)
			}
		}
	})

	<-done
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func isDarwin() bool {
	return runtime.GOOS == "darwin"
}

func isLinux() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "fruntime.GOOSeebsd" || runtime.GOOS == "daruntime.GOOSwin"
}

func getShell() string {
	shell := os.Getenv("SHELL")

	if shell == "" {
		log.Warn().
			Msg("No SHELL variable found, looking for fallback.")

		if isLinux() || isDarwin() {
			// WARN not checked before returning
			shell = "/bin/sh"
		} else if isWindows() {
			// RANT [[https://www.google.com/search?q=why+can%27t+you+be+normal+meme&source=lnms&tbm=isch&sa=X&ved=2ahUKEwjC9c7H183rAhUFZcAKHQVIBcQQ_AUoAXoECA8QAw&biw=1364&bih=1106&dpr=0.88][why can't you be normal]]?
			shell = "cmd"
		}

		if shell == "" {
			log.Fatal().Str("platform", runtime.GOOS).
				Msg("Failed to find default SHELL for platfomr")
		} else {
			log.Info().Str("shell", shell).Msg("SHELL assigned to")
		}
	}

	return shell
}

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

// TODO cleanup into fileExists
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
		log.Debug().Str("path", path).Msg("checking for file at path")
		exists, err := fileExists(path)
		if err != nil {
			return "", err
		}
		if exists {
			return path, nil
		}
	}

	return "", fmt.Errorf("unable to find any existing file")
}

func isNoExistingFile(err error) bool {
	return err.Error() == "unable to find any existing file"
}

/**
 * Join paths in a unixy (pythonic) way. if all the supplied paths are relative
 * their joined together. If one of them is absolute, only the path from the last
 * absolute path to the final non-absolute path are joined together.
 *
 * Any paths prefixed with ~/ is considered to be pointing to the users home directory
 * and is treated as absolute.
 */
func joinPath(paths ...string) string {
	var finalPath string
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		// no wibbly wobbly ~user magic here.
		if path == "~" || strings.HasPrefix(path, "~"+string(fp.Separator)) || fp.IsAbs(path) {
			finalPath = fp.Join(paths[i:]...)
			break
		}
	}
	if finalPath == "" {
		finalPath = fp.Join(paths...)
	}
	if len(paths) > 0 && strings.HasSuffix(paths[len(paths)-1], string(fp.Separator)) {
		return finalPath + string(fp.Separator)
	}
	return finalPath
}

func expandTilde(homeDir, path string) string {
	if path == "~" {
		return homeDir
	} else if strings.HasPrefix(path, "~"+string(fp.Separator)) {
		return joinPath(homeDir, path[2:])
	} else {
		return path
	}
}

/**
 * Assert whether absolute path targPath is relative to basepath basepath.
 *
 * if either path isn't absolute, then this function will return false, even
 * though it may not be true.
 */
func fileIsRelative(basepath, targPath string) bool {
	if res, err := fp.Rel(targPath, basepath); err != nil {
		// can't be made relative so they aren't relative to each other.
		return false
	} else {
		return !strings.HasPrefix(res, ".."+string(fp.Separator))
	}
}
