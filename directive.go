package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

type any = interface{}
type anySlice = []any

// Represents a single action that dotty can perform.
// This can involve linking a file, making a directory, etc.
type directive interface {
	// run directive
	run()

	// print directive in human readable
	log() string
}

type directiveConstructor = func(ctx *Context, args anySlice)

var directives map[edn.Keyword]directiveConstructor

func initDirectives() {
	directives = map[edn.Keyword]directiveConstructor{
		edn.Keyword("import"):   dImport,
		edn.Keyword("mkdir"):    dMkdir,
		edn.Keyword("mkdirs"):   dMkdir,
		edn.Keyword("link"):     dLink,
		edn.Keyword("shell"):    dShell,
		edn.Keyword("clean"):    dClean,
		edn.Keyword("when"):     dWhen,
		edn.Keyword("debug"):    dDebug,
		edn.Keyword("info"):     dInfo,
		edn.Keyword("warn"):     dWarn,
		edn.Keyword("def"):      dDef,
		edn.Keyword("package"):  dPackage,
		edn.Keyword("packages"): dPackage,
		edn.Keyword("ignore"):   dIgnore,
	}
}

/**
 * find the directive constructor associated with directive and initialise
 * it with the given arguments and context.
 */
func parseDirective(directive edn.Keyword, ctx *Context, args anySlice) {
	if init, ok := directives[directive]; ok {
		init(ctx, args)
	} else {
		log.Error().Str("directive", directive.String()).
			Interface("args", args).
			Msg("failed to find directive")
	}
}

/**
 * Given a list of directives of the same form as a dotty config file,
 * evaluate the parse out each directive and pass it to ParseDirective.
 */
func dispatchDirectives(ctx *Context, directives anySlice) {
	for i, directive := range directives {
		dir, ok := directive.(anySlice)
		if !ok {
			log.Error().Str("arg", fmt.Sprintf("%v", directive)).
				Msgf("Directives must be a list, not %T", directive)
			return
		}

		if len(dir) == 0 {
			log.Warn().Int("index", i+1).
				Msg("Empty directive found.")
			continue
		}

		dirKey, args := dir[0], dir[1:]
		if dirKey, ok := dirKey.(edn.Keyword); ok {
			if !ctx.skipDirectivePredicate(string(dirKey)) {
				parseDirective(dirKey, ctx, args)
			}
		} else {
			log.Warn().Int("index", i+1).
				Interface("value", dir).
				Msg("Directive statements should be keywords")
		}
	}
}

// The rest of this file just consists of utility functions to help make parsing
// fields out of simply AnyMaps a lot more straightforward.

// assert whether the conditions specified by opts evaluate to
// true or not.
//
// These conditions are the same as a general purpose :when directive.
// opts can also include a :if-bots directive which is just a shortcut
// for `:when (:bots ARGS)`, because that's most likely what this is
// going to be used for.
func directiveMapCondition(ctx *Context, opts map[any]any) bool {
	res := true
	if bots, ok := opts[edn.Keyword("if-bots")]; ok {
		if botsStr, ok := bots.(string); ok {
			res = dConditionInstallingBots(ctx, anySlice{botsStr})
		} else if botsSlice, ok := bots.(anySlice); ok {
			res = dConditionInstallingBots(ctx, botsSlice)
		} else {
			res = false
		}
	}

	if when, ok := opts[edn.Keyword("when")]; res && ok {
		res = dCondition(ctx, when)
	}

	return res
}

// read the boolean value name from ctx or opts into field, assigning a
// default value of def.
func readMapOptionBool(ctxOpts map[string]any, opts map[any]any, field *bool, name string, def bool) bool {
	*field = def // assign default

	opt, ok := ctxOpts[name]
	// override value from context with value from map (when provided).
	if optVal, optOk := opts[edn.Keyword(name)]; optOk {
		opt = optVal
		ok = true
	}
	if ok {
		if optBool, ok := opt.(bool); ok {
			*field = optBool // update value
		} else {
			log.Warn().Msgf("%s should be a boolean value, not %T", name, opt)
			return false
		}
	}
	return true
}

// same as readMapOptionBool but for strings. once we get generics we can
// abstract this away ლ(╹◡╹ლ).
func readMapOptionString(ctxOpts map[string]any, opts map[any]any, field *string, name string, def string) bool {
	*field = def // assign default

	opt, ok := ctxOpts[name]
	// override value from context with value from map (when provided).
	if optVal, optOk := opts[edn.Keyword(name)]; optOk {
		opt = optVal
		ok = true
	}
	if ok {
		if optString, ok := opt.(string); ok {
			*field = optString // update value
		} else {
			log.Warn().Msgf("%s should be a boolean value, not %T", name, opt)
			return false
		}
	}
	return true
}
