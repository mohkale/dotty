package main

import (
	"fmt"
	"testing"

	"olympos.io/encoding/edn"
)

func pathsFromEdn(data string) []interface{} {
	var paths []interface{}
	if edn.Unmarshal([]byte(data), &paths) != nil {
		panic(fmt.Sprintf("failed to parse edn: %s", data))
	}
	return paths
}

func identStr(a string) (string, bool) {
	return a, true
}

/**
 * run recursive build path on data and assert that we get exactly
 * expected back as a sequence of directories.
 */
func runRecursiveBuildPathOrderTest(t *testing.T, data string, expected []string) {
	t.Logf("working with data: %s", data)
	paths := pathsFromEdn(data)
	ch := make(chan string)
	go recursiveBuildPath(ch, paths, "", identStr, func(_ string, arg interface{}) {})

	for _, expected := range expected {
		actual := <-ch
		t.Logf("built path: %s", actual)
		if actual != expected {
			t.Errorf("yield mismatch: actual != expected, %v != %v", actual, expected)
		}
	}

	for v := range ch {
		t.Errorf("channel values not exhausted: %s", v)
	}
}

func TestRecursiveBuildPath_ReturnsArgsAsIs(t *testing.T) {
	// TODO maybe check whether the file path separator causes an error with tests windows
	runRecursiveBuildPathOrderTest(t,
		"(\"foo/bar\" \"foo\" \"bar\")",
		[]string{"foo/bar", "foo", "bar"},
	)

	runRecursiveBuildPathOrderTest(t,
		"(\"foo\" (\"bar\") \"baz\")",
		[]string{"foo", "bar", "baz"},
	)
}

func TestRecursiveBuildPath_IsRecursive(t *testing.T) {
	runRecursiveBuildPathOrderTest(t,
		"(\"foo\" (\"bar\" \"bag\" (\"hello\" \"world\" (\"friend\"))) \"baz\")",
		[]string{"foo", "bar/hello/friend", "bar/world/friend", "bag/hello/friend", "bag/world/friend", "baz"},
	)

	runRecursiveBuildPathOrderTest(t,
		"((\"foo\" (\"bar\") \"baz\" (\"bag\")))",
		[]string{"foo/bar", "foo/bag", "baz/bag"},
	)
}

func TestRecursiveBuildPath_RecursiveNilReferencesParent(t *testing.T) {
	runRecursiveBuildPathOrderTest(t,
		"((\"foo\" \"bar\" (nil)))",
		[]string{"foo", "bar"},
	)

	runRecursiveBuildPathOrderTest(t,
		"((nil (\"foo\" \"bar\")))",
		[]string{"foo", "bar", ""},
	)

	runRecursiveBuildPathOrderTest(t,
		"((\"foo\" \"bar\" (nil (\"baz\" \"bag\"))))",
		[]string{"foo/baz", "foo/bag", "bar/baz", "bar/bag"},
	)

	runRecursiveBuildPathOrderTest(t,
		"((\"foo\" \"bar\" \"baz\"))",
		[]string{"foo", "bar", "baz"},
	)
}

func TestRecursiveBuildPath_CallsErrorCallback(t *testing.T) {
	errorCalled := 0
	expected := 4
	paths := pathsFromEdn("(\"foo bar\" {} 5 ({} 5))")

	ch := make(chan string)
	go recursiveBuildPath(ch, paths, "", identStr, func(_ string, arg interface{}) {
		errorCalled += 1
	})
	for range ch {
	}

	if errorCalled != expected {
		t.Errorf("errorCallback called %d times, exected %d times", errorCalled, expected)
	}
}

func TestJoinPaths(t *testing.T) {
	testCases := []struct {
		paths  []string
		result string
	}{
		// empty paths list gives empty path
		{[]string{}, ""},
		// joining relative paths gives relative path
		{[]string{"foo", "bar", "baz"}, "foo/bar/baz"},
		// absolute path overrides any earlier paths
		{[]string{"foo", "/bar", "baz"}, "/bar/baz"},
		// home shortcut is treated as an absolute path
		{[]string{"foo", "~/bar", "baz"}, "~/bar/baz"},
		// only the last absolute path takes affect
		{[]string{"/foo", "/bar", "/baz"}, "/baz"},
		{[]string{"/foo", "/bar", "baz"}, "/bar/baz"},
		{[]string{"~/foo", "~/bar", "~/baz"}, "~/baz"},
		{[]string{"~/foo", "~/bar", "baz"}, "~/bar/baz"},
		// tilde, even by itself, is considered absolute
		{[]string{"~/foo", "~", "baz"}, "~/baz"},
		// trailing slashes aren't automatically stripped
		{[]string{"~/foo/"}, "~/foo/"},
	}

	for _, test := range testCases {
		res := joinPath(test.paths...)
		if res != test.result {
			t.Error("result mismatch: expected != actual", test.result, res)
		}
	}
}
