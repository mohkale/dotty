package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	fp "path/filepath"

	"github.com/mohkale/dotty/pkg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

type logErrorHook struct {
	callback func()
}

// call h.callback() if this logs level is at least as bad as an error.
func (h logErrorHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level >= zerolog.ErrorLevel {
		h.callback()
	}
}

func startDotty(opts *Options) *pkg.Context {
	ctx := pkg.CreateContext()
	ctx.Root = opts.RootDir
	ctx.Home = opts.HomeDir
	ctx.Cwd = ctx.Root
	ctx.OnlyDirectives = opts.OnlyDirectives.GetValues()
	ctx.ExceptDirectives = opts.ExceptDirectives.GetValues()
	ctx.Bots = opts.Bots.GetValues()
	ctx.Shell = pkg.GetShell()
	os.Setenv("HOME", opts.HomeDir)

	env := opts.EnvConfig
	if env == "" {
		// WARN ignoring error while looking for optional file, if you
		// want to explicitly make sure it's found, pass as a flag
		log.Debug().Str("cwd", ctx.Cwd).Msg("looking for env config")
		exists, _ := pkg.FindExistingFile(
			pkg.JoinPath(ctx.Root, ".dotty.env.edn"),
			pkg.JoinPath(ctx.Root, ".dotty.env"),
			pkg.JoinPath(ctx.Root, ".dotty"))
		env = exists
	}
	if env != "" {
		log.Info().
			Str("path", env).
			Msg("Importing environment file")

		pkg.LoadEdnSlice(env, func(env pkg.AnySlice) {
			pkg.ParseDirective(edn.Keyword("def"), ctx, env)
		})
	}

	go func() {
		defer close(ctx.DirChan)
		pkg.ParseDirective(edn.Keyword("import"), ctx, pkg.AnySlice{"config"})
	}()

	return ctx
}

func main() {
	cmd, opts := ParseArgs()

	initLogger(opts)

	// if an error is logged, program exits non-0.
	ok, errorHook := true, logErrorHook{}
	errorHook.callback = func() { ok = false }
	log.Logger = log.Hook(errorHook)

	switch cmd {
	case "install":
		ctx := startDotty(opts)
		for dir := range ctx.DirChan {
			dir.Run()
		}
		if opts.SaveBots != "" {
			saveBots(pkg.ExpandTilde(opts.HomeDir, pkg.JoinPath(opts.RootDir, opts.SaveBots)), ctx.Bots)
		}
	case "inspect":
		for dir := range startDotty(opts).DirChan {
			fmt.Println(dir.Log())
		}
	case "list-dirs":
		for key := range pkg.Directives {
			fmt.Println(string(key))
		}
	case "list-bots":
		bots := make(map[string]struct{})
		pkg.DConditionInstallingBots = func(ctx *pkg.Context, args pkg.AnySlice) bool {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					if _, ok := bots[argStr]; !ok {
						fmt.Println(argStr)
						bots[argStr] = struct{}{}
					}
				}
			}
			return true
		}
		for range startDotty(opts).DirChan {
		}
	default:
		fmt.Fprintf(os.Stderr, "%s error: unknown command: %s", PROG_NAME, cmd)
		os.Exit(1)
	}

	if !ok {
		os.Exit(1)
	}
}

// append the currently installing bots to the csv file at path
//
func saveBots(path string, bots []string) {
	dirname := fp.Dir(path)
	log.Debug().Str("path", dirname).
		Msg("Creating directory for bots file")
	if err := os.MkdirAll(dirname, 0744); err != nil {
		log.Fatal().Str("path", dirname).
			Err(err).
			Msg("Failed to create directory for bots file")
		return
	}

	log.Debug().Str("path", path).
		Msg("Checking whether bots file already exists")
	exists, err := pkg.PathExistsCheck(path, func(fi os.FileInfo) bool { return !fi.IsDir() }, true, true)
	if err != nil {
		log.Fatal().Str("path", path).
			Err(err).
			Msg("Failed to check whether bots file exists")
	} else if exists {
		log.Info().Str("path", path).
			Msg("Existing bots file found, opening it")
		fd, err := os.Open(path)
		if err != nil {
			log.Error().Str("path", path).
				Err(err).
				Msg("Failed to open bots file for writing")
			return
		}
		r := csv.NewReader(fd)
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal().Str("path", path).
					Err(err).
					Msg("Failed to parse bots file")
			}

			for _, bot := range record {
				if !pkg.StringSliceContains(bots, bot) {
					bots = append(bots, bot)
				}
			}
		}
		if err := fd.Close(); err != nil {
			log.Fatal().Str("path", path).
				Err(err).
				Msg("Failed to close opened bots file")
		}
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal().Str("path", path).
			Err(err).
			Strs("bots", bots).
			Msg("Failed to open bots file for writing bots")
	}
	defer fd.Close()
	w := csv.NewWriter(fd)
	if err := w.WriteAll([][]string{bots}); err != nil {
		log.Fatal().Str("path", path).
			Err(err).
			Strs("bots", bots).
			Msg("Failed to write bots to bots file")
	}
}
