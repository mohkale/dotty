package main

import (
	"fmt"
	"io/ioutil"
	"os"
	path "path/filepath"

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

// psuedo directive to import (one or more) configuration files.
func dImport(ctx *Context, args AnySlice) {
	if len(args) == 0 {
		log.Warn().Msg("Tried to import with no files")
		return
	}

	pathChan := make(chan string) // read paths to import into here.
	go recursiveBuildPath(pathChan, args, "", ctx.eval, func(base string, pathArg Any) {
		log.Error().
			Str("arg", fmt.Sprintf("%s", pathArg)).
			Str("base", base).
			Str("location", "parse").
			Msg("Encountered unacceptable type in import declaration")
		log.Fatal().Msg("Import targets can only be file paths or lists of paths")
	})

	for filepath := range pathChan {
		file, err := resolveImport(ctx.cwd, filepath)
		if err != nil {
			log.Error().
				Str("path", filepath).
				Str("cwd", ctx.cwd).
				Msg(err.Error())
			continue
		}

		log.Info().Str("path", file).Msg("Importing config file")
		LoadEdnSlice(file, func(conf AnySlice) {
			DispatchDirectives(ctx.chdir(path.Dir(file)), conf)
		})
	}
}

// given a cwd and a target import, try to find a file that matches
// the lookup rules for an import config and return it.
//
// if no file can be found or there was an error while checking for
// a file, return an error.
func resolveImport(cwd, target string) (string, error) {
	log.Debug().Str("cwd", cwd).
		Str("target", target).
		Msg("Looking for import target")

	targets := []string{
		// these really should be lazy, but go doesn't really have
		// a nice way of doing that... maybe channels.
		joinPath(cwd, target, "dotty.edn"),
		joinPath(cwd, target+".dotty"),
		joinPath(cwd, target+".edn"),
		joinPath(cwd, "."+target+".edn"),
		joinPath(cwd, "."+target),
		joinPath(cwd, target, ".config"), // short and sweet
		joinPath(cwd, target),
	}

	targetFile, err := findExistingFile(targets...)
	if err != nil {
		if isNoExistingFile(err) {
			return "", fmt.Errorf("Failed to resolve import target")
		} else {
			return "", err
		}
	}

	return targetFile, err
}
