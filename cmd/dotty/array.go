package main

import "strings"

type arrayFlags struct {
	metavar string
	values  []string
}

func (i *arrayFlags) String() string {
	return strings.Join(i.values, ",")
}

func (i *arrayFlags) Set(value string) error {
	i.values = append(i.values, value)
	return nil
}

func (i *arrayFlags) Type() string {
	if i.metavar == "" {
		return "arg"
	}
	return i.metavar
}

func (i *arrayFlags) GetValues() []string {
	return i.values
}
