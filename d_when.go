package main

import "github.com/rs/zerolog/log"
import "olympos.io/encoding/edn"

func dWhen(ctx *Context, args AnySlice) {
	if len(args) <= 1 {
		log.Warn().Msgf("Encountered %s directive with no body", edn.Keyword("when"))
		return
	}

	if dCondition(ctx, args[0]) {
		DispatchDirectives(ctx, args[1:])
	}
}

// makes conditional subcommands silent (they don't output anything) by default.
var dConditionDefaultCmdOpts = map[string]bool{"interactive": false, "quiet": true}

func dConditionPrepareCmd(opts map[Any]Any) map[Any]Any {
	for key, value := range dConditionDefaultCmdOpts {
		if _, ok := opts[edn.Keyword(key)]; !ok {
			opts[edn.Keyword(key)] = value
		}
	}
	return opts
}

/**
 * for building conditional statements. A condition is either:
 *  shell command (as accepted by dShell).
 *   "stat ./bar" or ("foo=./bar" "stat $bar")
 *
 *  The negation of a condition:
 *   (:not CONDITION)
 *
 *  An assertion, such as we're installing this bot:
 *   (:bot "git")
 */
func dCondition(ctx *Context, arg Any) bool {
	res := false
	assignRes := func(dir *shellDirective) { res = dir.exec() }

	if cmdOpts, ok := arg.(map[Any]Any); ok {
		dShellMappedCommand(ctx, dConditionPrepareCmd(cmdOpts), assignRes)
	} else if cmdSlice, ok := arg.(AnySlice); ok {
		if len(cmdSlice) == 0 {
			return false
		}

		if modifier, ok := cmdSlice[0].(edn.Keyword); ok {
			switch modifier {
			case edn.Keyword("not"):
				return !dCondition(ctx, cmdSlice[1:])
			case edn.Keyword("bots"):
				fallthrough
			case edn.Keyword("bot"):
				return dConditionInstallingBots(ctx, cmdSlice[1:])
			case edn.Keyword("and"):
				for _, cmd := range cmdSlice[1:] {
					if !dCondition(ctx, cmd) {
						return false
					}
				}
				return true
			case edn.Keyword("or"):
				for _, cmd := range cmdSlice[1:] {
					if dCondition(ctx, cmd) {
						return true
					}
				}
				return false
			default:
				log.Warn().Interface("condition", modifier).
					Msg("Unknown condition in when directive")
				return res
			}
		} else {
			dShellMappedCommand(ctx, dConditionPrepareCmd(map[Any]Any{edn.Keyword("cmd"): cmdSlice}), assignRes)
		}
	} else {
		dShellMappedCommand(ctx, dConditionPrepareCmd(map[Any]Any{edn.Keyword("cmd"): arg}), assignRes)
	}

	return res
}

var dConditionInstallingBots = func(ctx *Context, args AnySlice) bool {
	if len(args) == 0 {
		return false
	}

	for _, arg := range args {
		if bot, ok := arg.(string); ok {
			log.Trace().Str("bot", bot).Msg("Checking if installing bot")
			if !ctx.installingBot(bot) {
				return false
			}
		} else {
			log.Error().Str("bot", bot).
				Msgf("%s predicate can only accept strings, not %T", edn.Keyword("bot"), bot)
			return false
		}
	}

	return true
}
