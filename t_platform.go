package main

func _tPlatform(predicate func() bool) func(AnySlice) (AnySlice, error) {
	return func(dir AnySlice) (AnySlice, error) {
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
