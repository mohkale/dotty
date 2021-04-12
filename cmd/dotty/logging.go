package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type loggingLevel zerolog.Level

var loggingLevels = map[string]loggingLevel{
	"debug":    loggingLevel(zerolog.DebugLevel),
	"info":     loggingLevel(zerolog.InfoLevel),
	"warn":     loggingLevel(zerolog.WarnLevel),
	"error":    loggingLevel(zerolog.ErrorLevel),
	"fatal":    loggingLevel(zerolog.FatalLevel),
	"disabled": loggingLevel(zerolog.NoLevel),
	"trace":    loggingLevel(zerolog.TraceLevel),
}

func (level *loggingLevel) String() string {
	for key, value := range loggingLevels {
		if value == *level {
			return key
		}
	}
	return "UNKNOWN"
}

func (level *loggingLevel) Set(arg string) error {
	if newLevel, ok := loggingLevels[strings.ToLower(arg)]; ok {
		*level = newLevel
		return nil
	}

	options := ""
	i := 0
	for key := range loggingLevels {
		if i != 0 {
			options += ", "
		}
		options += key
		i++
	}
	return fmt.Errorf("unknown log level, choose one of: %s", options)
}

func (level *loggingLevel) Type() string {
	return "LEVEL"
}

func initLogger(opts *Options) {
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
			fmt.Fprintf(os.Stderr, "%s error: failed to open log file: %s", PROG_NAME, err.Error())
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
