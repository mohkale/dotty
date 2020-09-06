package main

import (
	"olympos.io/encoding/edn"
)

func InitTags() {
	edn.AddTagFn("dot/if-windows", tWindows)
	edn.AddTagFn("dot/if-linux", tLinux)
	edn.AddTagFn("dot/if-darwin", tDarwin)
}
