package main

import "runtime"

/**
 * Assert whether dotty is currently running in a windows system.
 */
func isWindows() bool {
	return runtime.GOOS == "windows"
}

/**
 * Assert whether dotty is currently running in a macos/darwin system.
 */
func isDarwin() bool {
	return runtime.GOOS == "darwin"
}

/**
 * Assert whether dotty is currently running in a linux system.
 */
func isLinux() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "fruntime.freebsd"
}

/**
 * Assert whether dotty is currently running in a unix like system.
 */
func isUnixy() bool {
	return isLinux() || isDarwin()
}

// return the flags to pass to the shell to run a cmd
//
// this tries to get around annoying errors when using
// windows cmd.exe as well.
func shellExecFlag(shell string) string {
	switch shell {
	case "cmd":
		fallthrough
	case "cmd.exe":
		return "/c"
	default:
		return "-c"
	}
}
