package pkg

import (
	"os"
	"runtime"

	"github.com/rs/zerolog/log"
)

func GetShell() string {
	shell := os.Getenv("SHELL")

	if shell == "" {
		log.Warn().
			Msg("No SHELL variable found, looking for fallback.")

		if isLinux() || isDarwin() {
			// WARN not checked before returning
			shell = "/bin/sh"
		} else if isWindows() {
			// RANT [[https://www.google.com/search?q=why+can%27t+you+be+normal+meme&source=lnms&tbm=isch&sa=X&ved=2ahUKEwjC9c7H183rAhUFZcAKHQVIBcQQ_AUoAXoECA8QAw&biw=1364&bih=1106&dpr=0.88][why can't you be normal]]?
			shell = "cmd"
		}

		if shell == "" {
			log.Fatal().Str("platform", runtime.GOOS).
				Msg("Failed to find default SHELL for platfomr")
		} else {
			log.Info().Str("shell", shell).Msg("SHELL assigned to")
		}
	}

	return shell
}
