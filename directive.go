package main

import "fmt"
import "github.com/rs/zerolog/log"
import "olympos.io/encoding/edn"

type Any = interface{}
type AnySlice = []Any

/**
 * Represents a single action that dotty can perform.
 * This can involve linking a file, making a directory, etc.
 */
type Directive interface {
	/** run directive */
	run()

	/**
	 * print directive in human readable form as a series
	 * of actions that the directive will run.
	 */
	log() string
}

type directiveConstructor = func(ctx *Context, args AnySlice)

var directives map[edn.Keyword]directiveConstructor

func InitDirectives() {
	directives = map[edn.Keyword]directiveConstructor{
		edn.Keyword("import"): dImport,
		edn.Keyword("mkdir"):  dMkdir,
		edn.Keyword("link"):   dLink,
		edn.Keyword("shell"):  dShell,
		edn.Keyword("clean"):  dClean,
		edn.Keyword("when"):   dWhen,
		edn.Keyword("debug"):  dDebug,
		edn.Keyword("info"):   dInfo,
		edn.Keyword("warn"):   dWarn,
		edn.Keyword("def"):    dDef,
		edn.Keyword("ignore"): dIgnore,
	}
}

/**
 * find the directive constructor associated with directive and initialise
 * it with the given arguments and context.
 */
func ParseDirective(directive edn.Keyword, ctx *Context, args AnySlice) {
	if init, ok := directives[directive]; ok {
		init(ctx, args)
	} else {
		log.Error().Str("directive", directive.String()).
			Str("args", fmt.Sprintf("%s", args)).
			Msg("failed to find directive")
	}
}

/**
 * Given a list of directives of the same form as a dotty config file,
 * evaluate the parse out each directive and pass it to ParseDirective.
 */
func DispatchDirectives(ctx *Context, directives AnySlice) {
	for i, directive := range directives {
		dir, ok := directive.(AnySlice)
		if !ok {
			log.Error().Str("arg", fmt.Sprintf("%s", directive)).
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
			ParseDirective(dirKey, ctx, args)
		} else {
			log.Warn().Int("index", i+1).
				Str("value", fmt.Sprintf("%v", dir)).
				Msg("Directive statements should be keywords")
		}
	}
}
