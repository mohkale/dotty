package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

type mkdirDirective struct {
	/** The path to the directory that is to be made */
	path string

	/** file permissions for the directory */
	chmod os.FileMode
}

func dMkdir(ctx *Context, args AnySlice) {
	recursiveBuildDirectivesFromPaths(ctx, args,
		func(ctx *Context, path string) {
			ctx.dirChan <- (&mkdirDirective{path: expandTilde(ctx.home, path)}).init(ctx)
		},
		func(opts map[Any]Any) (Any, bool) {
			src, ok := opts[edn.Keyword("path")]
			return src, ok
		},
		func(ctx *Context, opts map[Any]Any) *Context {
			if perms, ok := opts[edn.Keyword("chmod")]; ok {
				if permInt, err := strconv.ParseInt(fmt.Sprintf("%v", perms), 8, 64); err == nil {
					ctx.mkdirOpts["permissions"] = os.FileMode(permInt)
				} else {
					log.Warn().Str("permissions", fmt.Sprintf("%v", perms)).
						Msgf("permissions must be a valid file permission flag, not %T", perms)
				}
			}
			return ctx
		},
	)
}

func (dir *mkdirDirective) init(ctx *Context) *mkdirDirective {
	if dirPerms, ok := ctx.mkdirOpts["permissions"]; ok {
		dir.chmod = dirPerms.(os.FileMode)
	} else {
		// TODO get default permissions from fs
		dir.chmod = os.FileMode(0744)
	}

	return dir
}

func (dir *mkdirDirective) log() string {
	return fmt.Sprintf("mkdir %d %v", dir.chmod, dir.path)
}

func (dir *mkdirDirective) run() bool {
	log.Debug().Str("path", dir.path).
		Int("permissions", int(dir.chmod)).
		Msg("Creating directory")
	// TODO maybe skip if directory exists
	err := os.MkdirAll(dir.path, dir.chmod)
	if err != nil {
		log.Error().Str("path", dir.path).
			Int("permissions", int(dir.chmod)).
			Str("error", err.Error()).
			Msg("Failed to create directory")
		return false
	}
	return true
}
