package main

import (
	"testing"

	"olympos.io/encoding/edn"
)

func TestGenBotsGeneratesBots(t *testing.T) {
	paths := []string{"foo", "bar"}
	args := pathsFromEdn("(:import \"foo\" \"bar\")")
	out, err := tGenBots(args)

	if err != nil {
		t.Errorf("Tag failed: %s", err)
	}

	if args[0] != edn.Keyword("import") {
		t.Errorf("Tag doesn't have propper directive: expected != actual, %s != %s",
			edn.Keyword("import"), args[0])
	}

	if len(paths) != len(out)-1 {
		t.Errorf("Tag outputted different number of paths to input: expected != actual, %d != %d",
			len(paths), len(out)-1)
	}

	for i, obj := range out[1:] {
		obj := obj.(map[Any]Any)
		path := obj[edn.Keyword("path")].(string)
		if path != paths[i] {
			t.Errorf("Path value mismatch: expected != actual, %s != %s",
				paths[i], path)
		}

		if _, ok := obj[edn.Keyword("if-bots")]; !ok {
			t.Errorf("Path map doesn't have a %s property: %s", edn.Keyword("if-bots"),
				obj)
		}
	}
}

func TestGenBotsGeneratesTagsWithMap(t *testing.T) {
	args := pathsFromEdn("(:import {:path \"foo\" :foo true})")
	out, err := tGenBots(args)

	if err != nil {
		t.Errorf("Tag failed: %s", err)
	}

	res := out[1].(map[Any]Any)
	if res[edn.Keyword("path")] != "foo" {
		t.Errorf("Path mismatch, expected != actual, %s != %s",
			"foo", res[edn.Keyword("path")])
	}

	if _, ok := res[edn.Keyword("foo")]; !ok {
		t.Errorf("Result erased existing property foo: %s", res)
	}

	if _, ok := res[edn.Keyword("if-bots")]; !ok {
		t.Errorf("Path map doesn't have a %s property: %s", edn.Keyword("if-bots"), res)
	}
}

func TestGenBotsDoesntOverrideExistingIfBots(t *testing.T) {
	args := pathsFromEdn("(:import {:path \"foo\" :if-bots \"bar\"})")
	out, err := tGenBots(args)

	if err != nil {
		t.Errorf("Tag failed: %s", err)
	}

	res := out[1].(map[Any]Any)
	if res[edn.Keyword("path")] != "foo" {
		t.Errorf("Path mismatch, expected != actual, %s != %s",
			"foo", res[edn.Keyword("path")])
	}

	if bots, ok := res[edn.Keyword("if-bots")]; !ok {
		t.Errorf("Path map doesn't have a %s property: %s", edn.Keyword("if-bots"), res)
	} else if bots != "bar" {
		t.Errorf("Tag overrides existing :if-bots property: expected != actual, %s != %s",
			"bar", bots)
	}
}
