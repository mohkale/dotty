package main

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// generate a directive constructor that logs output to logFunc
func dLog(logFunc func() *zerolog.Event) directiveConstructor {
	return func(ctx *Context, args AnySlice) {
		if len(args) == 0 {
			return
		}

		if template, ok := args[0].(string); ok {
			logFunc().Msgf(template, args[1:]...)
		} else {
			log.Warn().Str("format", fmt.Sprintf("%s", args[0])).
				Msgf("Log functions first argument must always be a format string, not %T", args[0])
		}
	}
}

// directives for logging at different levels.
var (
	dDebug = dLog(log.Debug)
	dInfo  = dLog(log.Info)
	dWarn  = dLog(log.Warn)
)
