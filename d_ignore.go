package main

import "olympos.io/encoding/edn"

// Return the arguments of a directive that does nothing
func ignoreDirective() anySlice {
	return anySlice{edn.Keyword("ignore")}
}

// constructure for a directive that does nothing
func dIgnore(ctx *Context, args anySlice) {}
