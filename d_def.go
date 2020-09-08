package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// our approach here is a little complex because we allow the user to supply
// env key->value pairs as either a sequence of strings (in which case their)
// pushed to the env map. Or as sub lists of the same with a beginning keyword
// indicating their destination. For the case of supplying :env opts in a list
// you can't recurse down to another level. So, this is valid:
//   (:def (:env "foo" "bar")) // equivalent to (:def "foo" "bar")
// but this is not:
//   (:def (:env (:mkdir "foo" "bar")))
//
// To get this quirky syntax to work, while being as DRY as possible, we depend
// on callbacks and error handlers. The first run through uses assignEnvOpt to
// set environment variables and any keys which don't match the format get passed
// to an error handler which:
// - if the argument is a list such as (:env "foo" "bar" "baz"), it invokes this
//   function again while changing the error handler to actually cancel parsing
//   (because of the above condition).
// - if it's a list pointing to a map for a directive, it does the same as above.
// - otherwise log an error and skip forward.
//
// This function has a maximum recursive depth of 1, so there should be little overhead
// in practice.

// Pseudo directive for assigning options in the current context.
func dDef(ctx *Context, args anySlice) {
	var assignEnvOpt = func(key string, val any) {
		valString := fmt.Sprintf("%s", val)
		log.Debug().Str("key", key).
			Str("val", valString).
			Msg("Setting environment key with value")
		ctx.envOpts[key] = valString
		ctx.invalidateEnv()
	}

	var keyTypeError = func(key any) {
		log.Warn().Interface("key", key).
			Msgf(":def keys must be strings, not %T", key)
	}

	dDefDirectiveOpts(ctx, args, assignEnvOpt,
		func(key any) {
			args, ok := key.(anySlice)
			if !ok {
				keyTypeError(key)
				return
			}

			if len(args) == 0 {
				log.Warn().Msgf("%s entries must specify at least directive to configure.", edn.Keyword("def"))
			}

			dest, ok := args[0].(edn.Keyword)
			if !ok {
				log.Warn().Interface("key", args[0]).
					Msgf(":def directive keys must be EDN symbols, not %T", args[0])
				return
			}

			if dest == edn.Keyword("env") {
				dDefDirectiveOpts(ctx, args[1:], assignEnvOpt, keyTypeError)
			} else if destMap, ok := ctx.optsFromString(string(dest)); ok {
				dDefDirectiveOpts(ctx, args[1:], func(key string, value any) {
					log.Debug().Str("key", key).
						Str("val", fmt.Sprintf("%s", value)).
						Str("directive", string(dest)).
						Msgf("Setting key to value in options for %s", dest)
					destMap[key] = value
					ctx.invalidateEnv()
				}, keyTypeError)
			} else {
				log.Error().Str("directive", string(dest)).
					Msg("Unable to find configuration hash for directive")
			}
		})
}

// helper for dDef which reads each argument for args, if
// the argument is a string read the next argument as it's
// value and pass both to callback. Otherwise invoke errHandler.
func dDefDirectiveOpts(ctx *Context, args anySlice, callback func(string, any), errHandler func(any)) {
	for i := 0; i < len(args); i++ {
		key := args[i]
		if keyStr, ok := key.(string); ok {
			if i == len(args)-1 {
				log.Error().Str("key", keyStr).
					Msgf("%s directive encountered key with no associated value", edn.Keyword("def"))
				continue
			}
			i++
			callback(keyStr, args[i])
		} else {
			errHandler(key)
		}
	}
}
