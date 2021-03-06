package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	fp "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// load an edn slice of directives from the file at fpath and pass the
// result to callback.
func LoadEdnSlice(fpath string, callback func(AnySlice)) {
	fd, err := os.Open(fpath)
	if err != nil {
		log.Fatal().Str("path", fpath).
			Str("error", err.Error()).
			Msg("Failed to open file for reading")
	}
	defer fd.Close()

	iStream, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Fatal().Str("path", fpath).
			Str("err", err.Error()).
			Msg("Failed to read from file")
	}

	var conf AnySlice
	if err := edn.Unmarshal(iStream, &conf); err != nil {
		log.Fatal().Str("path", fpath).
			Str("err", err.Error()).
			Msg("Failed to parse file")
	}

	callback(conf)
}

// pseudo directive to import (one or more) configuration files.
func dImport(ctx *Context, args AnySlice) {
	if len(args) == 0 {
		log.Warn().Msg("Tried to import with no files")
		return
	}

	recursiveBuildDirectivesFromPaths(ctx, args,
		func(ctx *Context, filepath string) {
			file, err := resolveImport(filepath)
			if err != nil {
				log.Error().Str("path", filepath).
					Str("cwd", ctx.Cwd).
					Msg(err.Error())
				return
			}

			if StringSliceContains(*ctx.imports, file) {
				log.Warn().Str("path", file).Msg("Skipping import because it's already been imported")
			} else {
				*ctx.imports = append(*ctx.imports, file)

				log.Info().Str("path", file).Msg("Importing config file")
				LoadEdnSlice(file, func(conf AnySlice) {
					dispatchDirectives(ctx.chdir(fp.Dir(file)), conf)
				})
			}
		},
		func(opts map[Any]Any) (Any, bool) {
			src, ok := opts[edn.Keyword("path")]
			return src, ok
		},
		func(ctx *Context, opts map[Any]Any) (*Context, bool) {
			if !directiveMapCondition(ctx, opts) {
				return ctx, false
			}

			return ctx, true
		},
	)
}

// given a target path , try to find a file that matches
// the lookup rules for an import config and return it.
//
// if no file can be found or there was an error while checking for
// a file, return an error.
func resolveImport(target string) (string, error) {
	directory := fp.Dir(target)
	basename := fp.Base(target)
	log.Debug().Str("cwd", directory).
		Str("target", basename).
		Msg("Looking for import target")

	targets := []string{
		// these really should be lazy, but go doesn't really have
		// a nice way of doing that... maybe channels.
		JoinPath(directory, basename, "dotty.edn"),
		JoinPath(directory, basename+".dotty"),
		JoinPath(directory, basename+".edn"),
		JoinPath(directory, "."+basename+".edn"),
		JoinPath(directory, "."+basename),
		JoinPath(directory, basename, ".config"), // short and sweet
		JoinPath(directory, basename),
	}

	targetFile, err := FindExistingFile(targets...)
	if err != nil {
		if isNoExistingFile(err) {
			return "", fmt.Errorf("Failed to resolve import target")
		}

		return "", err
	}

	return targetFile, err
}
