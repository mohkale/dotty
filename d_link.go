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

func dLinkGeneratePaths(ctx *Context, arg Any, logTitle string) ([]string, bool) {
	if str, ok := arg.(string); ok {
		return []string{joinPath(ctx.cwd, str)}, true
	} else if slice, ok := arg.(AnySlice); ok {
		ch, paths := make(chan string), make([]string, 0)
		go recursiveBuildPath(ch, slice, ctx.cwd, ctx.eval, func(_ string, arg Any) {
			log.Fatal().Str("spec", fmt.Sprintf("%s", arg)).
				Str("path", fmt.Sprintf("%s", arg)).
				Msgf("link paths must be a string or a list of strings, not %T", arg)
		})
		for path := range ch {
			paths = append(paths, path)
		}
		return paths, true
	}

	log.Error().Str("src", fmt.Sprintf("%s", arg)).
		Msgf("%s must be a path, or a list of paths, not %T", logTitle, arg)
	return nil, false
}

func dLink(ctx *Context, args AnySlice) {
	for i := 0; i < len(args); i++ {
		path := args[i]

		if pathMap, ok := path.(map[Any]Any); ok {
			srcArg, ok := pathMap[edn.Keyword("src")]
			if !ok {
				log.Error().Str("spec", fmt.Sprintf("%s", path)).
					Msgf("link directive must specify a %s", edn.Keyword("src"))
				continue
			}

			src, ok := dLinkGeneratePaths(ctx, srcArg, "src")
			if !ok {
				continue
			}

			destArg, ok := pathMap[edn.Keyword("dest")]
			if !ok {
				log.Error().Str("spec", fmt.Sprintf("%s", path)).
					Msgf("link directive must specify a %s", edn.Keyword("dest"))
				continue
			}

			dest, ok := dLinkGeneratePaths(ctx, destArg, "dest")
			if !ok {
				continue
			}

			ctx.dirChan <- (&linkDirective{src: src, dest: dest}).init(ctx, pathMap)
		} else {
			if i == len(args)-1 {
				log.Error().Str("path", fmt.Sprintf("%s", path)).
					Msg("link: src with no dest encountered")
				continue
			}

			i++
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
 *
 * RANT GIVE ME GENERICS, plz
 */
func (dir *linkDirective) init(ctx *Context, opts map[Any]Any) *linkDirective {
	for _, slice := range [][]string{dir.src, dir.dest} {
		for i := range slice {
			slice[i] = expandTilde(ctx.home, slice[i])
		}
	}

	dir.mkdirs = true
	mkdirs, ok := ctx.linkOpts["mkdirs"]
	if optMkdir, optOk := opts[edn.Keyword("mkdirs")]; optOk {
		ok = true
		mkdirs = optMkdir
	}
	if ok {
		if mkdirs, ok := mkdirs.(bool); ok {
			dir.mkdirs = mkdirs
		} else {
			log.Warn().Msgf("mkdir should be a boolean value, not %T", mkdirs)
		}
	}

	dir.relink = false
	relink, ok := ctx.linkOpts["relink"]
	if optRelink, optOk := opts[edn.Keyword("relink")]; optOk {
		ok = true
		relink = optRelink
	}
	if ok {
		if relink, ok := relink.(bool); ok {
			dir.relink = relink
		} else {
			log.Warn().Msgf("mkdir should be a boolean value, not %T", mkdirs)
		}
	}

	dir.force = false
	force, ok := ctx.linkOpts["force"]
	if optForce, optOk := opts[edn.Keyword("force")]; optOk {
		ok = true
		force = optForce
	}
	if ok {
		if force, ok := force.(bool); ok {
			dir.force = force
		} else {
			log.Warn().Msgf("mkdir should be a boolean value, not %T", mkdirs)
		}
	}

	dir.glob = false
	glob, ok := ctx.linkOpts["glob"]
	if optGlob, optOk := opts[edn.Keyword("glob")]; optOk {
		ok = true
		glob = optGlob
	}
	if ok {
		if glob, ok := glob.(bool); ok {
			dir.glob = glob
		} else {
			log.Warn().Msgf("mkdir should be a boolean value, not %T", mkdirs)
		}
	}

	dir.ignoreMissing = false
	ignoreMissing, ok := ctx.linkOpts["ignore-missing"]
	if optIgnoreMissing, optOk := opts[edn.Keyword("ignore-missing")]; optOk {
		ok = true
		ignoreMissing = optIgnoreMissing
	}
	if ok {
		if ignoreMissing, ok := ignoreMissing.(bool); ok {
			dir.ignoreMissing = ignoreMissing
		} else {
			log.Warn().Msgf(":ignore-missing should be a boolean value, not %T", mkdirs)
		}
	}

	dir.symbolic = true
	symbolic, ok := ctx.linkOpts["symbolic"]
	if optSymbolic, optOk := opts[edn.Keyword("symbolic")]; optOk {
		ok = true
		symbolic = optSymbolic
	}
	if ok {
		if symbolic, ok := symbolic.(bool); ok {
			dir.symbolic = symbolic
		} else {
			log.Warn().Msgf("mkdir should be a boolean value, not %T", mkdirs)
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

func (dir *linkDirective) run() bool {
	if dir.glob || len(dir.src) > 1 {
		// linking multiple files into one (or more) destinations. Make sure
		// each destination has a trailing slash to indicate it's a directory.
		for i := 0; i < len(dir.dest); i++ {
			// WARN this'll probably fail if you're using windows style paths.
			// TODO fix
			if !strings.HasSuffix(dir.dest[i], string(fp.Separator)) {
				dir.dest[i] += string(fp.Separator)
			}
		}
	}

	srcCh := make(chan string)
	go dir.linkSources(srcCh)

	for src := range srcCh {
		for _, dest := range dir.dest {
			if strings.HasSuffix(dest, string(fp.Separator)) {
				dest = joinPath(dest, fp.Base(src))
			}

			destInfo, err := os.Lstat(dest)
			destExists := true
			// TODO some heavy refactoring
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

	return false
}

func (dir *linkDirective) linker() func(string, string) error {
	if dir.symbolic {
		return os.Symlink
	} else {
		return os.Link
	}
}

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
				// NOTE only symlinks can have missing sources, hardlinks require a
				// backing file.
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
