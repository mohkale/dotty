package pkg

import (
	"fmt"
	"olympos.io/encoding/edn"
)

func init() {
	tags := []struct {
		name string
		fun  func(AnySlice) (AnySlice, error)
	}{
		{"if-windows", tWindows},
		{"if-linux", tLinux},
		{"if-darwin", tDarwin},
		{"if-unix", tUnix},
		{"gen-bots", tGenBots},
		{"link-gen", tLinkGen},
	}
	for _, tag := range tags {
		if err := edn.AddTagFn("dot/"+tag.name, tag.fun); err != nil {
			panic(fmt.Sprintf("Failed to assign tag %s: %s", tag.name, err))
		}
	}
}
