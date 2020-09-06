package main

import "olympos.io/encoding/edn"

func ignoreDirective() AnySlice {
	return AnySlice{edn.Keyword("ignore")}
}

func tWindows(directive AnySlice) (AnySlice, error) {
	if isWindows() {
		return directive, nil
	}
	return ignoreDirective(), nil
}

func tLinux(directive AnySlice) (AnySlice, error) {
	if isLinux() {
		return directive, nil
	}
	return ignoreDirective(), nil
}

func tDarwin(directive AnySlice) (AnySlice, error) {
	if isDarwin() {
		return directive, nil
	}
	return ignoreDirective(), nil
}
