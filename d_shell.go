package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// A directive for spawning and running shell commands
type shellDirective struct {
	// the command line to exec. this should be a string of the form
	// the you can pass to sh -c.
	cmd string

	// current working directory in which the command should run
	cwd string

	// environment variables (of a form compatible with exec.Spawn)
	env []string

	// the shell in which to run the command.
	shell string

	// an optional message to be outputted (at INFO level) before
	// running cmd.
	desc string

	// suppress logging related to command (including exit status)
	quiet bool

	// toggle the default values of stdin, stdout, stderr
	interactive bool

	// let command read from standard input.
	stdin bool

	// let command write to stdout
	stdout bool

	// let command write to stderr
	stderr bool
}

func dShell(ctx *Context, args AnySlice) {
	callback := func(dir *shellDirective) {
		ctx.dirChan <- dir
	}
	for _, cmd := range args {
		dShellCommand(ctx, cmd, callback)
	}
}

type dShellPreparedCallback = func(dir *shellDirective)

func dShellCommand(ctx *Context, cmd Any, onDone dShellPreparedCallback) bool {
	if opts, ok := cmd.(map[Any]Any); ok {
		return dShellMappedCommand(ctx, opts, onDone)
	} else {
		return dShellListCommand(ctx, cmd, onDone)
	}
}

/**
 * For when the command line is being supplied as a map of options
 * alongside the command line. This has been abstracted into it's
 * own function because you can't recursively change the options for
 * a command line after part of it has already been provided.
 */
func dShellMappedCommand(ctx *Context, opts map[Any]Any, onDone dShellPreparedCallback) bool {
	cmd, ok := opts[edn.Keyword("cmd")]
	if !ok {
		log.Error().Interface("opts", opts).
			Msgf("shell directive must supply a %s field", edn.Keyword("cmd"))
		return false
	}

	if cmdStr, ok := cmd.(string); ok {
		onDone((&shellDirective{cmd: cmdStr, shell: ctx.shell}).init(ctx, opts))
		return true
	} else {
		newCtx := ctx.clone()
		for key, val := range opts {
			key, ok := key.(edn.Keyword)
			if key == "cmd" || !ok {
				continue
			}

			newCtx.shellOpts[string(key)] = val
		}

		return dShellListCommand(newCtx, cmd, onDone)
	}
}

/**
 * Construct a shell directive from a shell command line or a
 * list of such command lines. Like dShellMappedCommand this has
 * been abstracted into its own function because you can't recursively
 * build a command line... YET :grin:.
 */
func dShellListCommand(ctx *Context, cmd Any, onDone dShellPreparedCallback) bool {
	if cmdSlice, ok := cmd.(AnySlice); ok {
		var cmd string
		for i, line := range cmdSlice {
			if lineStr, ok := line.(string); ok {
				if i != 0 {
					cmd += "\n"
				}
				cmd += lineStr
			} else {
				log.Error().Interface("cmd", cmdSlice).
					Msgf("Shell command lines can only consist of strings, not %T", line)
				return false
			}
		}
		onDone((&shellDirective{cmd: cmd, shell: ctx.shell}).init(ctx, nil))
	} else if cmdStr, ok := cmd.(string); ok {
		onDone((&shellDirective{cmd: cmdStr, shell: ctx.shell}).init(ctx, nil))
	} else {
		log.Error().Interface("cmd-line", cmd).
			Msgf("shell command lines can only be lines, lists of lines or maps containing them, not %T", cmd)
		return false
	}
	return true
}

func (dir *shellDirective) init(ctx *Context, opts map[Any]Any) *shellDirective {
	dir.env = ctx.environ()
	dir.cwd = ctx.cwd

	readMapOptionString(ctx.shellOpts, opts, &dir.desc, "desc", "")
	readMapOptionBool(ctx.shellOpts, opts, &dir.interactive, "interactive", false)
	readMapOptionBool(ctx.shellOpts, opts, &dir.quiet, "quiet", false)
	readMapOptionBool(ctx.shellOpts, opts, &dir.stdin, "stdin", dir.interactive)
	readMapOptionBool(ctx.shellOpts, opts, &dir.stdout, "stdout", dir.interactive)
	readMapOptionBool(ctx.shellOpts, opts, &dir.stderr, "stderr", dir.interactive)

	return dir
}

func (dir *shellDirective) log() string {
	return fmt.Sprintf("shell %s", dir.cmd)
}

func (dir *shellDirective) run() {
	dir.exec()
}

func buildCommand(cmdLine []string, cwd string, env []string, stdin bool, stdout bool, stderr bool) *exec.Cmd {
	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	cmd.Dir = cwd
	cmd.Env = env

	if stdout {
		cmd.Stdout = os.Stdout
	}
	if stderr {
		cmd.Stderr = os.Stderr
	}
	if stdin {
		cmd.Stdin = os.Stdin
	}

	return cmd
}

func (dir *shellDirective) exec() bool {
	cmd := buildCommand([]string{dir.shell, shellExecFlag(dir.shell), dir.cmd},
		dir.cwd, dir.env, dir.stdin, dir.stdout, dir.stderr)

	if dir.desc != "" {
		log.Info().Msg(dir.desc)
	}

	if !dir.quiet {
		log.Debug().Str("shell", dir.shell).
			Str("cmd", dir.cmd).
			Bool("interactive", dir.interactive).
			Bool("quiet", dir.quiet).
			Msg("running subcommand")
	}

	if err := cmd.Run(); err != nil {
		if !dir.quiet {
			if exitErr, ok := err.(*exec.ExitError); ok {
				log.Error().Str("shell", dir.shell).
					Str("cmd", dir.cmd).
					Int("code", exitErr.ExitCode()).
					Err(err).
					Msg("Subcommand failed")
			} else {
				log.Error().Str("shell", dir.shell).
					Str("cmd", dir.cmd).
					Err(err).
					Msg("Failed to spawn subcommand")
			}
		}
		return false
	}
	return true
}
