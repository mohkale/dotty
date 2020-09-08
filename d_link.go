package main

import (
	"fmt"
	"os"
	fp "path/filepath"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// A directive to link files from src into dest.
type linkDirective struct {
	src  []string
	dest []string

	/** make any parent directories for the link beforehand. */
	mkdirs bool

	/** if dest exists and is a symlink, overwrite it. */
	relink bool

	/** Overwrite existing dest, even none links. This implies relink. */
	force bool

	/** src is a list of glob paths, link all files matching glob into dest */
	glob bool

	/** if src is missing, make link anyway as a broken symlink */
	ignoreMissing bool

	/** make a symlink, not an hard link */
	symbolic bool
}

// generate the paths for a link (src or dest) from arg.
//
// NOTE we can't use recursiveBuildDirectivesFromPaths because srcs
// and destinations aren't grouped into a single arg. Their separated
// as "src" then "dest" which can appear as many times as desired.
//
// logTitle is used to let include which path type (src or dest) we're
// building in the logging output.
func dLinkGeneratePaths(ctx *Context, arg any, logTitle string) ([]string, bool) {
	if str, ok := arg.(string); ok {
		if str, ok = ctx.eval(str); ok {
			return []string{joinPath(ctx.cwd, fp.FromSlash(str))}, true
		}
	} else if slice, ok := arg.(anySlice); ok {
		ch, paths := make(chan string), make([]string, 0)
		go recursiveBuildPath(ch, slice, ctx.cwd, ctx.eval, func(_ string, arg any) {
			log.Fatal().Interface("spec", arg).
				Interface("path", arg).
				Msgf("Link paths must be a string or a list of strings, not %T", arg)
		})
		for path := range ch {
			paths = append(paths, path)
		}
		return paths, true
	}

	log.Error().Interface("src", arg).
		Msgf("%s must be a path, or a list of paths, not %T", logTitle, arg)
	return nil, false
}

// constructor for linkDirective
func dLink(ctx *Context, args anySlice) {
LoopStart:
	for i := 0; i < len(args); i++ {
		path := args[i]

		if pathMap, ok := path.(map[any]any); ok {
			if !directiveMapCondition(ctx, pathMap) {
				continue
			}

			// to avoid having to repeat the following logic twice, we abstract
			// the interface for dealing with both src and dest into two fields
			// in a struct.
			paths := []*struct {
				field string
				paths []string
			}{
				{"src", []string{}},
				{"dest", []string{}},
			}

			for _, path := range paths {
				arg, ok := pathMap[edn.Keyword(path.field)]
				if !ok {
					log.Error().Interface("spec", path).
						Msgf("Link directive must specify a %s", edn.Keyword(path.field))
					continue LoopStart
				}
				if paths, ok := dLinkGeneratePaths(ctx, arg, path.field); ok {
					path.paths = paths
				} else {
					continue LoopStart
				}
			}

			ctx.dirChan <- (&linkDirective{src: paths[0].paths, dest: paths[1].paths}).init(ctx, pathMap)
		} else {
			if i == len(args)-1 {
				log.Error().Interface("src", path).
					Msg("Link src with no destination encountered")
				continue
			}

			i++
			// NOTE cleaning up this duplication would take even more lines, so lets leave it.
			src, ok := dLinkGeneratePaths(ctx, path, "src")
			if !ok {
				continue
			}
			dest, ok := dLinkGeneratePaths(ctx, args[i], "dest")
			if !ok {
				continue
			}

			ctx.dirChan <- (&linkDirective{src: src, dest: dest}).init(ctx, nil)
		}
	}
}

/**
 * populate directive defaults from either the context or current options.
 */
func (dir *linkDirective) init(ctx *Context, opts map[any]any) *linkDirective {
	for _, slice := range [][]string{dir.src, dir.dest} {
		for i := range slice {
			slice[i] = expandTilde(ctx.home, slice[i])
		}
	}

	readMapOptionBool(ctx.linkOpts, opts, &dir.mkdirs, "mkdirs", true)
	readMapOptionBool(ctx.linkOpts, opts, &dir.relink, "relink", false)
	readMapOptionBool(ctx.linkOpts, opts, &dir.force, "force", false)
	readMapOptionBool(ctx.linkOpts, opts, &dir.glob, "glob", false)
	readMapOptionBool(ctx.linkOpts, opts, &dir.ignoreMissing, "ignore-missing", false)
	readMapOptionBool(ctx.linkOpts, opts, &dir.symbolic, "symbolic", true)

	// linking multiple files into one (or more) destinations. Make sure
	// each destination has a trailing slash to indicate it's a directory.
	if dir.glob || len(dir.src) > 1 {
		for i := 0; i < len(dir.dest); i++ {
			if !strings.HasSuffix(dir.dest[i], string(fp.Separator)) {
				dir.dest[i] += string(fp.Separator)
			}
		}
	}

	return dir
}

func (dir *linkDirective) log() string {
	var prefix string
	if dir.symbolic {
		prefix = "-s"
	} else {
		prefix = "-P"
	}
	if dir.force {
		prefix += "f"
	}
	if dir.glob {
		prefix = "glob " + prefix
	}
	var res string
	for i, src := range dir.src {
		for j, dest := range dir.dest {
			if i != 0 || j != 0 {
				res += "\n"
			}
			res += fmt.Sprintf("link %s %s %s", prefix, src, dest)
		}
	}
	return res
}

func (dir *linkDirective) run() {
	srcCh := make(chan string)
	go dir.linkSources(srcCh)

	// TODO some heavy refactoring. There's a lot of edge cases here
	// so it's easier to keep it all in one place, but this should really
	// be broken down.
	for src := range srcCh {
		for _, dest := range dir.dest {
			if strings.HasSuffix(dest, string(fp.Separator)) {
				dest = joinPath(dest, fp.Base(src))
			}

			destInfo, err := os.Lstat(dest)
			destExists := true
			if err != nil {
				if os.IsNotExist(err) {
					destExists = false
				} else {
					errno := err.(*os.PathError).Err.(syscall.Errno)
					if errno == syscall.ENOTDIR {
						destExists = false
					}
					log.Error().Str("path", dest).
						Str("error", err.Error()).
						Msg("Failed to stat destination")
				}
			}

			if destExists {
				if dir.force || (dir.relink && destInfo.Mode()&os.ModeSymlink != 0) {
					if destInfo.IsDir() {
						// it's not safe to recursively delete a directory and replace
						// it with a symlink.
						log.Warn().Str("src", src).
							Str("dest", dest).
							Msg("Skipping force link because dest is a directory")
						continue
					}
					if err := os.Remove(dest); err != nil {
						log.Error().Str("src", src).
							Str("dest", dest).
							Str("error", err.Error()).
							Msg("Failed to remove dest before relink, skipping")
						continue
					}
				} else {
					if destInfo.IsDir() {
						dest = joinPath(dest, fp.Base(src))
					} else {
						// NOTE this has debug level because linking a file to a file that exists
						// is pretty common... I.E. when you're linking a file to the same file it's
						// already linked to.
						log.Debug().Str("src", src).
							Str("dest", dest).
							Msg("Skipping linking src to dest because dest exists.")
						continue
					}
				}
			} else {
				destParent := fp.Dir(dest)
				if destParentExists, err := dirExists(destParent); err != nil {
					log.Error().Str("src", src).
						Str("dest", dest).
						Str("destParent", destParent).
						Str("error", err.Error()).
						Msg("Failed to stat container for dest")
					continue
				} else if !destParentExists {
					if dir.mkdirs {
						// WARN hardcoded file permission
						if err := os.MkdirAll(destParent, 0744); err != nil {
							log.Error().Str("path", destParent).
								Msg("Failed to create parent directory for dest")
							continue
						}
					} else {
						log.Warn().Str("src", src).
							Str("dest", dest).
							Msg("Skipping link because destination parent doesn't exist")
						continue
					}
				}
			}

			log.Info().Str("src", src).
				Str("dest", dest).
				Msg("Linking src to dest")
			if err := dir.linker()(src, dest); err != nil {
				log.Error().Str("src", src).
					Str("dest", dest).
					Str("error", err.Error()).
					Msg("Failed to link files")
			}
		}
	}
}

// The function used to link this kind of directive (symbolic or hard link).
func (dir *linkDirective) linker() func(string, string) error {
	if dir.symbolic {
		return os.Symlink
	}

	return os.Link
}

// pass list of files to be linked from the sources for this
// directive into ch.
//
// This also expands any globs when dir.glob is true.
//
// WARN when expanding globs, there's a chance no files will
// be returned.
func (dir *linkDirective) linkSources(ch chan string) {
	defer close(ch)
	for _, src := range dir.src {
		if dir.glob {
			if globs, err := fp.Glob(src); err != nil {
				log.Error().Str("glob", src).
					Str("error", err.Error()).
					Msg("Glob failed")
			} else {
				for _, path := range globs {
					ch <- path
				}
			}
		} else {
			// we're linking from file to file, first check whether the file exists
			// or if we don't care, then return the file as is.
			if dir.symbolic && dir.ignoreMissing {
				ch <- src
			} else if exists, err := pathExists(src); err != nil {
				log.Error().Str("path", src).
					Str("error", err.Error()).
					Msg("Error when checking file exists")
			} else if exists {
				ch <- src
			} else {
				log.Error().Str("path", src).
					Msg("Link src not found")
			}
		}
	}
}
