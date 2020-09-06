package cli

import (
	"fmt"

	"github.com/rs/zerolog"
)

type loggingLevel zerolog.Level

var loggingLevels = map[string]loggingLevel{
	"debug":    loggingLevel(zerolog.DebugLevel),
	"info":     loggingLevel(zerolog.InfoLevel),
	"warn":     loggingLevel(zerolog.WarnLevel),
	"error":    loggingLevel(zerolog.ErrorLevel),
	"fatal":    loggingLevel(zerolog.FatalLevel),
	"disabled": loggingLevel(zerolog.NoLevel),
}

func (level *loggingLevel) String() string {
	for key, value := range loggingLevels {
		if value == *level {
			return key
		}
	}
	return "UNKNOWN"
}

func (level *loggingLevel) Set(arg string) error {
	if newLevel, ok := loggingLevels[arg]; ok {
		*level = newLevel
		return nil
	} else {
		options := ""
		i := 0
		for key := range loggingLevels {
			if i != 0 {
				options += ", "
			}
			options += key
			i++
		}
		return fmt.Errorf("unknown log level, choose one of: %s", options)
	}
}

func (level *loggingLevel) Type() string {
	return "LEVEL"
}
