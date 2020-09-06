package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type csvFlags struct {
	metavar string
	values  []string
}

func (f *csvFlags) String() string {
	return strings.Join(f.values, ",")
}

func (f *csvFlags) Set(arg string) error {
	r := csv.NewReader(strings.NewReader(arg))

	for {
		value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s error: failed to parse csv: %s", PROG_NAME, arg)
		}
		f.values = append(f.values, value...)
	}

	return nil
}

func (i *csvFlags) Type() string {
	if i.metavar == "" {
		return "arg"
	}
	return i.metavar
}

func (i *csvFlags) GetValues() []string {
	return i.values
}
