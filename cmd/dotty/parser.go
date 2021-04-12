package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	fp "path/filepath"
	"strings"

	"github.com/rs/zerolog"
	flag "github.com/spf13/pflag"
)

// PROG_NAME The name of this program
var PROG_NAME = "dotty"

// The version of this program
var progVersion = "0.0.2"

var usage = fmt.Sprintf(`
%s is the delightfully lispy dotfile manager.
`, PROG_NAME)

// Options - store for parsed command line option data
type Options struct {
	LogLevel         loggingLevel
	LogFile          string
	LogJson          bool
	RootDir          string
	HomeDir          string
	EnvConfig        string
	OnlyDirectives   csvFlags
	ExceptDirectives csvFlags
	SaveBots         string
	Bots             csvFlags
}

func (opts *Options) init() *Options {
	opts.LogLevel = loggingLevel(zerolog.InfoLevel)
	opts.OnlyDirectives.metavar = "directive"
	opts.ExceptDirectives.metavar = "directive"
	return opts
}

func generateSubcommand(name string, do func(*flag.FlagSet, *Options)) func(*Options) *flag.FlagSet {
	return func(opts *Options) *flag.FlagSet {
		set := flag.NewFlagSet(PROG_NAME+" "+name, flag.ExitOnError)
		do(set, opts)
		return set
	}
}

func sharedConfigurationOpts(set *flag.FlagSet, opts *Options) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: failed to stat cwd", PROG_NAME)
		os.Exit(1)
	}

	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: failed to get current user info", PROG_NAME)
		os.Exit(1)
	}

	set.StringVarP(&opts.RootDir, "cd", "d", cwd, "chdir to here before processing")
	set.StringVarP(&opts.EnvConfig, "config", "c", "", "path to environment config. relative to rootdir.")
	set.StringVarP(&opts.HomeDir, "home", "H", user.HomeDir, "path to environment config. relative to rootdir.")
}

func sharedInstallationOpts(set *flag.FlagSet, opts *Options) {
	set.VarP(&opts.OnlyDirectives, "only", "o", "only run the supplied directives. this option overrides -e.")
	set.VarP(&opts.ExceptDirectives, "except", "e", "run any directives apart from these")
	set.VarP(&opts.Bots, "bots", "b", "specify subbots to invoke as csv. See README.")
}

var subCommands = map[string]struct {
	description string
	flagSet     func(*Options) *flag.FlagSet
}{
	"install": {
		"install dotfiles from configuration",
		generateSubcommand("install", func(set *flag.FlagSet, opts *Options) {
			sharedInstallationOpts(set, opts)
			sharedConfigurationOpts(set, opts)
			dottyBotsFile := ".dotty.bots"
			if envBots, ok := os.LookupEnv("DOTTY_BOTS_FILE"); ok {
				dottyBotsFile = envBots
			}
			set.StringVarP(&opts.SaveBots, "save-bots", "B", dottyBotsFile, "Append installing bots to this file. Set to empty to disable.")
		}),
	},
	"inspect": {
		"print actions in human readable form",
		generateSubcommand("inspect", func(set *flag.FlagSet, opts *Options) {
			sharedInstallationOpts(set, opts)
			sharedConfigurationOpts(set, opts)
		}),
	},
	"list-dirs": {
		"list all directives known to dotty",
		generateSubcommand("list-dirs", func(set *flag.FlagSet, opts *Options) {
		}),
	},
	"list-bots": {
		"list all bots in all imports from your root import",
		generateSubcommand("list-bots", func(set *flag.FlagSet, opts *Options) {
			sharedConfigurationOpts(set, opts)
		}),
	},
}

func subCommandKeys() []string {
	subCommandKeys := make([]string, len(subCommands))
	i := 0
	for key := range subCommands {
		subCommandKeys[i] = key
		i++
	}
	return subCommandKeys
}

func rootFlagSet(opts *Options) *flag.FlagSet {
	set := flag.NewFlagSet(PROG_NAME, flag.ExitOnError)
	flag.ErrHelp = errors.New("")

	set.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <OPTIONS> {%s}\n", PROG_NAME, strings.Join(subCommandKeys(), ","))
		fmt.Fprintln(os.Stderr, usage)
		fmt.Fprintln(os.Stderr, "Subcommands:")
		for sub, spec := range subCommands {
			fmt.Fprintf(os.Stderr, "  %s - %s\n", sub, spec.description)
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		set.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "%s, version %s\n", PROG_NAME, progVersion)
	}

	set.SetInterspersed(false)
	set.SortFlags = false
	set.VarP(&opts.LogLevel, "log-level", "l", "set verbosity of logging output")
	set.StringVarP(&opts.LogFile, "log-file", "L", "", "fd to dump logging output to. supply - to use stdout. (default stderr)")
	set.BoolVarP(&opts.LogJson, "log-json", "j", false, "output log file as JSON, not human readable")
	return set
}

// ParseArgs parse command line arguments into a command
// to execute and an options struct.
func ParseArgs() (string, *Options) {
	opts := (&Options{}).init()
	root := rootFlagSet(opts)

	root.Parse(os.Args[1:])

	subArgs := root.Args()
	if len(subArgs) == 0 {
		fmt.Fprintf(os.Stderr, "%s error: must supply a subcommand. one of: %s\n", PROG_NAME, strings.Join(subCommandKeys(), ","))
		os.Exit(1)
	}

	if subCmd, ok := subCommands[subArgs[0]]; ok {
		subCmd.flagSet(opts).Parse(subArgs[1:])
	} else {
		fmt.Fprintf(os.Stderr, "%s error: unknown subcommand: %s\n", PROG_NAME, subArgs[0])
		os.Exit(1)
	}

	if root, err := fp.Abs(opts.RootDir); err != nil {
		fmt.Fprintf(os.Stderr, "%s error: unable to find root directory: %s", PROG_NAME, opts.RootDir)
	} else {
		opts.RootDir = fp.FromSlash(root)
	}

	return subArgs[0], opts
}
