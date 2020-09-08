package main

import (
	"fmt"
	fs "path/filepath"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// generate a destination path from a src path
func tLinkGenGetDest(src Any) (Any, bool) {
	if srcStr, ok := src.(string); ok {
		dest := fs.Base(srcStr)
		if dest[0] != '.' {
			dest = "." + dest
		}
		return joinPath("~", dest), true
	} else if _, ok := src.(AnySlice); ok {
		return "~", true
	}

	log.Warn().Str("directive", fmt.Sprintf("%s", src)).
		Msg("link: unable to generate destination for directive")
	return nil, false
}

// generate a src path from a destination path
func tLinkGenGetSrc(dest Any) (Any, bool) {
	if srcStr, ok := dest.(string); ok {
		src := fs.Base(srcStr)
		if src[0] == '.' {
			src = src[1:]
		}

		if src == "" {
			log.Warn().Str("path", srcStr).
				Msg("src path resolved to an empty destination path")
			return nil, false
		}

		return src, true
	}

	log.Warn().Str("directive", fmt.Sprintf("%s", dest)).
		Msg("link: unable to generate src for directive")
	return nil, false
}

// A tag to automatically generate src or destination fields for a link
// directive.
//
// BUG(mohkale) evaluation of tags takes place before construction of Context
// so we're creating a dummy context here to evaluate any env variables in the
// paths. The only substitutions these can manage will be environment variables.
// Any changes made to the local context won't appear.
//
// A decent workaround would be to spread out recursively built paths, such as
// (("foo" ("bar" "baz"))) => (("foo" ("bar") ("foo" ("baz"))) but that would require
// reimplementing a good chunk of the logic in recursiveBuildPath to support chanelling
// String slices instead of strings. I'll wait until go adds generics to fix this, it's
// personally low priority.
//
// To summarise, we shouold go from:
// #dot/link-gen
// (:link
// 	"~/.foo"
// 	(".foo" ".bar" ".baz" (".bag"))
// 	("${XDG_CONFIG_HOME}"
// 		(".bar" ".baz" ".bag"))
// 	;; or dest if it's missing instead
//   {:src "local_file"
//    :mkdirs false}
//   ;; when both are present, #dot/link-gen is ignored.
//   {:src "local_file" :dest "dest_file"}
//   ;; more complex structures lead to more complex maps.
//   {:src ("local_file" ("foo" "bar"))})
//
// to:
// (:link ("$XDG_CONFIG_HOME" (".bar")) "bar"
//        ("$XDG_CONFIG_HOME" (".baz")) "baz"
//        ("$XDG_CONFIG_HOME" (".bag")) "bag"
//        {:src "local_file"
//         :dest "~/.local_file"
//         :mkdirs false}
//        {:src "local_file"
//         :dest "dest_file"}
//        {:src "local_file"
//         :dest "dest_file"}
//        {:src "local_file/foo"
//         :dest "~/.foo"}
//        {:src "local_file/bar"
//         :dest "~/.bar"})
//
func tLinkGen(args AnySlice) (AnySlice, error) {
	if len(args) == 0 {
		return args, nil
	}

	if args[0] != edn.Keyword("link") {
		log.Fatal().Str("directive", fmt.Sprintf("%s", args)).
			Msgf("the link-gen tag can only be applied to %s directives", edn.Keyword("link"))
	}

	newArgs := make(AnySlice, 0, len(args))
	newArgs = append(newArgs, edn.Keyword("link"))

	for _, path := range args[1:] {
		if pathMap, ok := path.(map[Any]Any); ok {
			src, srcOk := pathMap[edn.Keyword("src")]
			dest, destOk := pathMap[edn.Keyword("dest")]
			if !srcOk && !destOk {
				log.Fatal().Str("spec", fmt.Sprintf("%s", path)).
					Msgf("The gen-link tag requires either a %s or %s field for every spec",
						edn.Keyword("src"), edn.Keyword("dest"))
			}

			if !(srcOk && destOk) {
				if srcOk {
					if dest, ok := tLinkGenGetDest(src); ok {
						pathMap[edn.Keyword("dest")] = dest
					} else {
						continue
					}
				} else {
					if src, ok := tLinkGenGetSrc(dest); ok {
						pathMap[edn.Keyword("src")] = src
					} else {
						continue
					}
				}
			}

			newArgs = append(newArgs, pathMap)
		} else if paths, ok := dLinkGeneratePaths(&Context{}, AnySlice{path}, "dest"); ok {
			for _, dest := range paths {
				if src, ok := tLinkGenGetSrc(dest); ok {
					newArgs = append(newArgs, src)
					newArgs = append(newArgs, dest)
				}
			}
		}
	}

	return newArgs, nil
}
