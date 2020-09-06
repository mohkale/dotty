package main

import (
	"fmt"
	"io/ioutil"
	"os"
	fp "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

type cleanDirective struct {
	path string
	root string

	/** remove dead links, even if they don't point to a file in root. */
	force bool

	/** recursively search for deadlinks */
	recursive bool
}

func dClean(ctx *Context, args AnySlice) {
	recursiveBuildDirectivesFromPaths(ctx, args,
		func(ctx *Context, path string) {
			ctx.dirChan <- (&cleanDirective{path: expandTilde(ctx.home, path), root: ctx.root}).init(ctx)
		},
		func(opts map[Any]Any) (Any, bool) {
			src, ok := opts[edn.Keyword("path")]
			return src, ok
		},
		func(ctx *Context, opts map[Any]Any) *Context {
			if force, ok := opts[edn.Keyword("force")]; ok {
				if forceBool, ok := force.(bool); ok {
					ctx.cleanOpts["force"] = forceBool
				} else {
					log.Warn().Str("force", fmt.Sprintf("%v", force)).
						Msgf("The %s option must be a valid boolean, not %T", edn.Keyword("force"), force)
				}
			}

			if recursive, ok := opts[edn.Keyword("recursive")]; ok {
				if recursiveBool, ok := recursive.(bool); ok {
					ctx.cleanOpts["recursive"] = recursiveBool
				} else {
					log.Warn().Str("recursive", fmt.Sprintf("%v", recursive)).
						Msgf("The %s option must be a valid boolean, not %T", edn.Keyword("recursive"), recursive)
				}
			}

			return ctx
		},
	)
}

func (dir *cleanDirective) init(ctx *Context) *cleanDirective {
	if forceBool, ok := ctx.cleanOpts["force"]; ok {
		dir.force = forceBool.(bool)
	}

	if recursiveBool, ok := ctx.cleanOpts["recursive"]; ok {
		dir.recursive = recursiveBool.(bool)
	}

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
type FileInfoWithPath struct {
	info os.FileInfo
	path string
}

func (dir *cleanDirective) run() bool {
	fileCh := make(chan FileInfoWithPath)
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

		// make sure the link is relevent to dotty
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
	return false
}

func (dir *cleanDirective) getFiles(ch chan FileInfoWithPath) {
	if dir.recursive {
		err := fp.Walk(dir.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			ch <- FileInfoWithPath{info, path}
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
				ch <- FileInfoWithPath{info, joinPath(dir.path, info.Name())}
			}
		}
	}

	close(ch)
}
