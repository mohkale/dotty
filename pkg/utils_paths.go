package pkg

import (
	"fmt"
	fp "path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type recursiveBuildPathErrorCallback = func(base string, arg Any)

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
	defer close(ch)
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
					currentPaths[i] = JoinPath(base, fp.FromSlash(pathStr))
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
				ch <- JoinPath(base, fp.FromSlash(pathStr))
			}
		} else if pathSlice, ok := path.(AnySlice); ok {
			recursiveDo(pathSlice, base)
		} else {
			err(base, path)
		}
	}
}

func recursiveBuildPathIdentityPreJoin(a string) (string, bool) {
	return a, true
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
	updateContext func(ctx *Context, opts map[Any]Any) (*Context, bool),
) {
	pathCh, done := make(chan string), make(chan struct{})
	go func() {
		for path := range pathCh {
			pathCompleteCallback(ctx, path)
		}
		done <- struct{}{}
	}()

	go recursiveBuildPath(pathCh, args, ctx.Cwd, ctx.eval, func(base string, arg Any) {
		// the only situation in which the input can not be a path is when
		// it's a map containing perhaps more directories.
		sMap, ok := arg.(map[Any]Any)
		if !ok {
			log.Warn().Interface("directive", arg).
				Msgf("Directive must be a map of symbols to options, not %T", arg)
			return
		}

		if src, ok := getSrcsFromOpts(sMap); ok {
			newCtx, ok := updateContext(ctx.chdir(base), sMap)
			if !ok {
				return
			}
			if srcStr, ok := src.(string); ok {
				src = AnySlice{srcStr}
			}

			if srcSlice, ok := src.(AnySlice); ok {
				recursiveBuildDirectivesFromPaths(newCtx, srcSlice,
					pathCompleteCallback, getSrcsFromOpts, updateContext)
			} else {
				log.Warn().
					Str("path", fmt.Sprintf("%s", src)).
					Msgf("Path must be a string or list of strings, not %T", src)
			}
		}
	})

	<-done
}

func isNoExistingFile(err error) bool {
	return err.Error() == "Unable to find any existing file"
}

/**
 * Join paths in a unixy (pythonic) way. if all the supplied paths are relative
 * their joined together. If one of them is absolute, only the path from the last
 * absolute path to the final non-absolute path are joined together.
 *
 * Any paths prefixed with ~/ is considered to be pointing to the users home directory
 * and is treated as absolute.
 *
 * NOTE: Trailing slashes aren't stripped by this implementation
 *  joinPath("~/foo", "bar/") // => "~/foo/bar/"
 */
func JoinPath(paths ...string) string {
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

/**
 * Substitute ~/ for homeDir in path
 */
func ExpandTilde(homeDir, path string) string {
	if path == "~" {
		return homeDir
	} else if strings.HasPrefix(path, "~"+string(fp.Separator)) {
		return JoinPath(homeDir, path[2:])
	} else {
		return path
	}
}

/**
 * Assert whether absolute path targPath is relative to basepath basepath.
 *
 * if either path isn't absolute, then this function will return false, even
 * though the two paths may actually be relative.
 */
func fileIsRelative(basepath, targPath string) bool {
	if res, err := fp.Rel(targPath, basepath); err == nil {
		return !strings.HasPrefix(res, ".."+string(fp.Separator))
	}

	// can't be made relative so they aren't relative to each other.
	return false
}
