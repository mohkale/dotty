package main

import (
	fp "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// Wow we generate bot names for the import directives
func tGenBotsGetBotName(target string) string {
	return fp.Base(target)
}

// automatically generate a :if-bots entry for all
// import targets in the following :import directive.
func tGenBots(dir AnySlice) (AnySlice, error) {
	if len(dir) == 0 {
		return dir, nil
	}

	if dir[0] != edn.Keyword("import") {
		log.Fatal().Interface("directive", dir).
			Msg("The gen-bots tag can only be applied to :import directives")
	}

	newArgs := make(AnySlice, 0, len(dir))
	newArgs = append(newArgs, edn.Keyword("import"))

	pathCh := make(chan string)
	go recursiveBuildPath(pathCh, dir[1:], "",
		recursiveBuildPathIdentityPreJoin,
		tGenBotsMapHandler(func(complete Any) {
			newArgs = append(newArgs, complete)
		}),
	)

	for path := range pathCh {
		newArgs = append(newArgs, map[Any]Any{
			edn.Keyword("path"):    path,
			edn.Keyword("if-bots"): tGenBotsGetBotName(path),
		})
	}

	return newArgs, nil
}

// Handle the user supplying the import target as a map.
// This isn't pretty, we need to account for the user supplying the
// import path as a string `(:import {:path "foo/bar"})` and as a
// slice of recursive paths `(:import {:path ("foo" "bar")})`.
func tGenBotsMapHandler(onComplete func(Any)) func(string, Any) {
	return func(base string, arg Any) {
		argMap, ok := arg.(map[Any]Any)
		if !ok {
			log.Fatal().Interface("arg", arg).
				Msg("Import arguments must be paths, lists of paths or maps containing paths")
		}

		path, ok := argMap[edn.Keyword("path")]
		if !ok {
			log.Warn().Interface("arg", arg).
				Msg("Import maps must specify a :path field")
			return
		}

		if pathStr, ok := path.(string); ok {
			// when paths a string, just assign the if-bots field if it's not assigned
			if _, ok := argMap[edn.Keyword("if-bots")]; !ok {
				argMap[edn.Keyword("if-bots")] = tGenBotsGetBotName(pathStr)
			}
			onComplete(argMap)
		} else if pathSlice := path.(AnySlice); ok {
			// recursively build all paths and create a copy of the current map
			// for each of them.
			subPathCh := make(chan string)
			go recursiveBuildPath(subPathCh, pathSlice, base,
				recursiveBuildPathIdentityPreJoin,
				func(_ string, _ Any) {
					// because there's no way at the moment to inherit attributes down (without ctx).
					log.Warn().Msg("The gen-bots tag doesn't support map paths with map depth > 1")
				},
			)

			for path := range subPathCh {
				// copy the current map
				newMap := make(map[Any]Any)
				for key, val := range newMap {
					newMap[key] = val
				}

				// assign the path and if-bots properties
				newMap[edn.Keyword("path")] = path
				if _, ok := argMap[edn.Keyword("if-bots")]; !ok {
					argMap[edn.Keyword("if-bots")] = tGenBotsGetBotName(pathStr)
				}
				onComplete(newMap)
			}
		} else {
			onComplete(arg) // Let the import directive itself handle whatever this is.
		}
	}
}
