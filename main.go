package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	fp "path/filepath"

	"github.com/mohkale/dotty/cli"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

func initLogger(opts *cli.Options) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var writer io.Writer
	if opts.LogFile == "" {
		writer = os.Stderr
	} else if opts.LogFile == "-" {
		writer = os.Stdout
	} else {
		if file, err := os.OpenFile(opts.LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err == nil {
			writer = file
		} else {
			fmt.Fprintf(os.Stderr, "%s error: failed to open log file: %s", cli.PROG_NAME, err.Error())
			os.Exit(1)
		}
	}

	if opts.LogJson {
		log.Logger = log.Output(writer)
	} else {
		// open a console writer with color only when not writing to a file
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: writer,
			NoColor: !(opts.LogFile == "" || opts.LogFile == "-")})
	}

	zerolog.SetGlobalLevel(zerolog.Level(opts.LogLevel))
}

type logErrorHook struct {
	callback func()
}

// call h.callback() if this logs level is at least as bad as an error.
func (h logErrorHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level >= zerolog.ErrorLevel {
		h.callback()
	}
}

func startDotty(opts *cli.Options) *Context {
	ctx := createContext()
	ctx.root = opts.RootDir
	ctx.home = opts.HomeDir
	ctx.cwd = ctx.root
	ctx.onlyDirectives = opts.OnlyDirectives.GetValues()
	ctx.exceptDirectives = opts.ExceptDirectives.GetValues()
	ctx.bots = opts.Bots.GetValues()
	ctx.shell = getShell()
	os.Setenv("HOME", opts.HomeDir)

	env := opts.EnvConfig
	if env == "" {
		// WARN ignoring error while looking for optional file, if you
		// want to explicitly make sure it's found, pass as a flag
		log.Debug().Str("cwd", ctx.cwd).Msg("looking for env config")
		exists, _ := findExistingFile(
			joinPath(ctx.root, ".dotty.env.edn"),
			joinPath(ctx.root, ".dotty.env"),
			joinPath(ctx.root, ".dotty"))
		env = exists
	}
	if env != "" {
		log.Info().
			Str("path", env).
			Msg("Importing environment file")

		loadEdnSlice(env, func(env anySlice) {
			parseDirective(edn.Keyword("def"), ctx, env)
		})
	}

	go func() {
		defer close(ctx.dirChan)
		parseDirective(edn.Keyword("import"), ctx, anySlice{"config"})
	}()

	return ctx
}

func main() {
	cmd, opts := cli.ParseArgs()

	initLogger(opts)
	initDirectives()
	initTags()

	// if an error is logged, program exits non-0.
	ok, errorHook := true, logErrorHook{}
	errorHook.callback = func() { ok = false }
	log.Logger = log.Hook(errorHook)

	switch cmd {
	case "install":
		ctx := startDotty(opts)
		for dir := range ctx.dirChan {
			dir.run()
		}
		if opts.SaveBots != "" {
			saveBots(expandTilde(opts.HomeDir, joinPath(opts.RootDir, opts.SaveBots)), ctx.bots)
		}
	case "inspect":
		for dir := range startDotty(opts).dirChan {
			fmt.Println(dir.log())
		}
	case "list-dirs":
		for key := range directives {
			fmt.Println(string(key))
		}
	case "list-bots":
		bots := make(map[string]struct{})
		dConditionInstallingBots = func(ctx *Context, args anySlice) bool {
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
		for range startDotty(opts).dirChan {
		}
	default:
		fmt.Fprintf(os.Stderr, "%s error: unknown command: %s", cli.PROG_NAME, cmd)
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
	exists, err := _pathExists(path, func(fi os.FileInfo) bool { return !fi.IsDir() }, true, true)
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
				if !stringSliceContains(bots, bot) {
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
