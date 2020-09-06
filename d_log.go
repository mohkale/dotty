package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

/**
 * generate a directive constructor that logs output to logFunc
 */
func dLog(logFunc func() *zerolog.Event) directiveConstructor {
	return func(ctx *Context, args AnySlice) {
		if len(args) == 0 {
			return
		}

		if template, ok := args[0].(string); ok {
			logFunc().Msgf(template, args[1:]...)
		} else {
			log.Warn().
				Msgf("log functions first argument must always be a format string, not %T", args[0])
		}
	}
}

var dDebug = dLog(log.Debug)
var dInfo = dLog(log.Info)
var dWarn = dLog(log.Warn)
