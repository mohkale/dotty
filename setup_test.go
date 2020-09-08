package main

import (
	"fmt"
	"olympos.io/encoding/edn"
)

func pathsFromEdn(data string) anySlice {
	var paths []interface{}
	if edn.Unmarshal([]byte(data), &paths) != nil {
		panic(fmt.Sprintf("Failed to parse edn: %s", data))
	}
	return paths
}

func identStr(a string) (string, bool) {
	return a, true
}
