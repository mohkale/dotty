package main

func _tPlatform(predicate func() bool) func(anySlice) (anySlice, error) {
	return func(dir anySlice) (anySlice, error) {
		if predicate() {
			return dir, nil
		}
		return ignoreDirective(), nil
	}
}

var tWindows = _tPlatform(isWindows)
var tLinux = _tPlatform(isLinux)
var tDarwin = _tPlatform(isDarwin)
var tUnix = _tPlatform(isUnixy)
