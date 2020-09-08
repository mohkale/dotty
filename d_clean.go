package main

import (
	"fmt"
	"io/ioutil"
	"os"
	fp "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

/**
 * A directive for removing dead symlinks from paths in the file system.
 *
 * A symlink is dead if the file it points to doesn't exist. This directive
 * goes through all the files in path (recursively if the recursive option
 * is provided). Finds any that point to root and are dead-links and then
 * deletes them.
 *
 * With the force option it doesn't bother to check whether the symlink points
 * to some file in root.
 */
type cleanDirective struct {
	// The directory containing files to be cleaned
	path string

	// The root directory, any dead links must point to a child file
	// of this path (unless force is true).
	root string

	// remove dead links, even if they don't point to a file in root
	force bool

	// look in path and all valid subdirectories of path
	recursive bool
}

/**
 * constructor for cleanDirective.
 */
func dClean(ctx *Context, args anySlice) {
	recursiveBuildDirectivesFromPaths(ctx, args,
		// create and completed paths as a directive to dirChan
		func(ctx *Context, path string) {
			ctx.dirChan <- (&cleanDirective{path: expandTilde(ctx.home, path), root: ctx.root}).init(ctx)
		},
		// get new paths from the :path parameter when the argument is a map.
		func(opts map[any]any) (any, bool) {
			src, ok := opts[edn.Keyword("path")]
			return src, ok
		},
		// update context with opts
		func(ctx *Context, opts map[any]any) (*Context, bool) {
			if !directiveMapCondition(ctx, opts) {
				return ctx, false
			}

			// luckily all configurable fields are booleans so no reflection needed.
			for _, opt := range []string{"force", "recursive"} {
				if arg, ok := opts[edn.Keyword(opt)]; ok {
					if argBool, ok := arg.(bool); ok {
						ctx.cleanOpts[opt] = argBool
					} else {
						log.Warn().Interface("force", arg).
							Msgf("The %s option must be a valid boolean, not %T", edn.Keyword(opt), arg)
					}
				}
			}

			return ctx, true
		},
	)
}

// initialise a new directive instanec with options from the Context.
func (dir *cleanDirective) init(ctx *Context) *cleanDirective {
	readMapOptionBool(ctx.cleanOpts, nil, &dir.force, "force", false)
	readMapOptionBool(ctx.cleanOpts, nil, &dir.recursive, "recursive", false)
	return dir
}

func (dir *cleanDirective) log() string {
	var flags string
	if dir.force {
		flags += "-f "
	}
	if dir.recursive {
		flags += "-r "
	}
	return fmt.Sprintf("clean %s%s", flags, dir.path)
}

// RANT go really needs tuple types or offer a nicer way
// to return multiple values from a function.
type fileInfoWithPath struct {
	info os.FileInfo
	path string
}

func (dir *cleanDirective) run() {
	fileCh := make(chan fileInfoWithPath)
	go dir.getFiles(fileCh)
	for file := range fileCh {
		// we only clean symlinks, not regular files
		if file.info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		dest, err := os.Readlink(file.path)
		if err != nil {
			log.Error().Str("link", file.path).
				Str("error", err.Error()).
				Msg("Failed to readlink")
			continue
		}

		// make sure the link is relevant to dotty
		if !dir.force && !fileIsRelative(dest, dir.root) {
			continue
		}

		if exists, err := fileExists(dest); err != nil {
			log.Error().Str("path", dest).
				Str("error", err.Error()).
				Msg("Error when checking file exists")
		} else if !exists {
			log.Info().Str("path", file.path).Msg("Cleaning dead link")
			if err := os.Remove(file.path); err != nil {
				log.Error().Str("path", file.path).
					Str("error", err.Error()).
					Msg("Error when removing dead link")
			}
		}
	}
}

/**
 * channel the files this directive should consider for cleaning into ch.
 */
func (dir *cleanDirective) getFiles(ch chan fileInfoWithPath) {
	defer close(ch)
	if dir.recursive {
		err := fp.Walk(dir.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			ch <- fileInfoWithPath{info, path}
			return nil
		})
		if err != nil {
			log.Error().Str("path", dir.path).
				Str("error", err.Error()).
				Msg("Error while recursing path")
		}
	} else {
		if files, err := ioutil.ReadDir(dir.path); err != nil {
			log.Error().Str("path", dir.path).
				Str("error", err.Error()).
				Msg("Error while listing path")
		} else {
			for _, info := range files {
				ch <- fileInfoWithPath{info, joinPath(dir.path, info.Name())}
			}
		}
	}
}
