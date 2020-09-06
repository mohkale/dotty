package main

import (
	"fmt"
	"io/ioutil"
	"os"
	path "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

func loadEdnSlice(fpath string, callback func(AnySlice)) {
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

/**
 * Config import pseudo directive. It's a pseudo directive because
 * it doesn't return a directive to be evaluated. Instead it finishes
 * evaluation after being invoked and then exits.
 */
func dImport(ctx *Context, args AnySlice) {
	if len(args) == 0 {
		log.Warn().Msg("Tried to import with no files")
		return
	}

	pathChan := make(chan string)
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

		loadEdnSlice(file, func(conf AnySlice) {
			DispatchDirectives(ctx.chdir(path.Dir(file)), conf)
		})
	}
}

func resolveImport(cwd, target string) (string, error) {
	log.Debug().Str("cwd", cwd).
		Str("target", target).
		Msg("Looking for import target")

	targets := []string{
		joinPath(cwd, target, "dotty.edn"),
		joinPath(cwd, target+".dotty"),
		joinPath(cwd, target+".edn"),
		joinPath(cwd, "."+target+".edn"),
		joinPath(cwd, "."+target),
		joinPath(cwd, target, ".config"), // short and sweet
		joinPath(cwd, target),
	}

	// base := joinPath(cwd, target)
	// base2 := joinPath(base, ".config")
	// withExt := base + ".edn"
	// dirBasenameAsFile := path.Base(target)

	targetFile, err := findExistingFile(targets...)
	if err != nil {
		if isNoExistingFile(err) {
			return "", fmt.Errorf("failed to resolve import target")
		} else {
			return "", err
		}
	}

	return targetFile, err
}
