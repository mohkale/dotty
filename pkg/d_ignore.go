package pkg

import "olympos.io/encoding/edn"

// Return the arguments of a directive that does nothing
func ignoreDirective() AnySlice {
	return AnySlice{edn.Keyword("ignore")}
}

// constructure for a directive that does nothing
func dIgnore(ctx *Context, args AnySlice) {}
