package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// directive to create a directory
type mkdirDirective struct {
	// The path to the directory that is to be made
	path string

	// file permissions for the directory
	chmod os.FileMode
}

func dMkdir(ctx *Context, args anySlice) {
	recursiveBuildDirectivesFromPaths(ctx, args,
		// complete paths go into the directive channel
		func(ctx *Context, path string) {
			ctx.dirChan <- (&mkdirDirective{path: expandTilde(ctx.home, path)}).init(ctx)
		},
		// encountered a map, recurse into any further maps.
		func(opts map[any]any) (any, bool) {
			src, ok := opts[edn.Keyword("path")]
			return src, ok
		},
		// update context.
		func(ctx *Context, opts map[any]any) (*Context, bool) {
			if !directiveMapCondition(ctx, opts) {
				return ctx, false
			}

			if perms, ok := opts[edn.Keyword("chmod")]; ok {
				if permInt, err := strconv.ParseInt(fmt.Sprintf("%v", perms), 8, 64); err == nil {
					ctx.mkdirOpts["permissions"] = os.FileMode(permInt)
				} else {
					log.Warn().Str("permissions", fmt.Sprintf("%v", perms)).
						Msgf("Permissions must be a valid file permission flag, not %T", perms)
				}
			}
			return ctx, true
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

func (dir *mkdirDirective) run() {
	if exists, err := pathExists(dir.path, false); err != nil {
		log.Error().Str("path", dir.path).
			Str("error", err.Error()).
			Msg("Failed to check whether directory exists")
		return
	} else if exists {
		log.Debug().Str("path", dir.path).
			Msg("Skipping creating directory because path exists")
		return
	}

	log.Info().Str("path", dir.path).
		Int("permissions", int(dir.chmod)).
		Msg("Creating directory")
	err := os.MkdirAll(dir.path, dir.chmod)
	if err != nil {
		log.Error().Str("path", dir.path).
			Int("permissions", int(dir.chmod)).
			Str("error", err.Error()).
			Msg("Failed to create directory")
	}
}
