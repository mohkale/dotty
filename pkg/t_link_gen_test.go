package pkg

import (
	"os"
	fs "path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	code := m.Run()
	os.Exit(code)
}

func TestSrcFromDest_EmptyDestinationsArentReturned(t *testing.T) {
	if _, ok := tLinkGenGetSrc("."); ok {
		t.Error("tLinkGenGetSrc is ok with empty src path: .")
	}
}

func TestDestFromSrc_FilesAreMovedToHomeWithHiddenPrefix(t *testing.T) {
	testCases := []struct {
		src  string
		dest string /* basename */
	}{
		{"foo", ".foo"},
		{".bar", ".bar"},
		{"~/foo/bar/baz", ".baz"},
	}

	for _, test := range testCases {
		res, ok := tLinkGenGetDest(test.src)
		if !ok {
			t.Errorf("get-dest returned not ok with path: %s", test.src)
			continue
		}

		dest := res.(string)
		dirname, basename := fs.Dir(dest), fs.Base(dest)
		if basename != test.dest {
			t.Errorf("destination doesn't match file with hidden prefix: expected != actual, %s != %s",
				test.dest, basename)
		}

		if dirname != "~" {
			t.Error("destination isn't placed in the home directory")
		}
	}
}

func TestDestFromSrc_CollectionsAreMovedToHome(t *testing.T) {
	paths := AnySlice{"foo", "bar", "baz"}
	dest, ok := tLinkGenGetDest(paths)
	if !ok {
		t.Errorf("get-dest returned not ok with paths: %s", paths)
		return
	}

	if dest != "~" {
		t.Errorf("result mismatch: expected != actual, %s != %s", "~", dest)
	}
}

// TODO test actual tag instead of just utils
