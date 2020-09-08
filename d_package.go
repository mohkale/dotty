package main

import (
	"strings"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

// A directive for installing packages using a package manager.
type packageDirective struct {
	// the name of the package being installed
	pkg string

	// the command line used to install the package
	cmd []string

	managerName string
	manager     *packageManager

	// these fields are taken from shellDirective and have the same meaning
	cwd         string
	env         []string
	interactive bool
	stdin       bool
	stdout      bool
	stderr      bool

	// manual installation command as shell script, this will override cmd
	manual *shellDirective

	// run before installation.
	// NOTE if this fails, then the installation step will be skipped.
	before *shellDirective

	// run after installation has finished.
	// NOTE this only runs if the installation process exited properly.
	after *shellDirective
}

func dPackageDefaultHandler(ctx *Context, args AnySlice) {
	log.Warn().
		Msgf("No package manager found, running default clause.")
	dShell(ctx, args)
}

func dPackage(ctx *Context, args AnySlice) {
	for _, arg := range args {
		argSlice, ok := arg.(AnySlice)

		if !ok {
			log.Warn().Interface("arg", arg).
				Msgf("The :package directive must be supplied arguments of the form (:package \"spec\"), not %T", arg)
			continue
		}

		if len(argSlice) == 0 {
			continue
		}

		pacman, ok := argSlice[0].(edn.Keyword)
		if !ok {
			log.Warn().Interface("pacman", pacman).
				Msgf("Package managers must be symbols, not %T", pacman)
			continue
		}

		if pacman == edn.Keyword("default") {
			dPackageDefaultHandler(ctx, argSlice[1:])
			return
		}

		manager, ok := packageManagers[pacman]
		if !ok {
			log.Error().Interface("pacman", pacman).
				Msgf("Unknown package manager")
			continue
		}

		// check whether this manager exists on the system
		if manager.execPath == "" {
			manager.execPath = manager.exists()
			if manager.execPath == "" {
				log.Debug().Str("pacman", string(pacman)).
					Msg("Failed to find package manager")
				continue
			}
		}

		// build directives for each package target
		for _, pkg := range argSlice[1:] {
			dPackageBuildDirective(ctx, manager, string(pacman), pkg)
		}

		return // package manager found, cancel check
	}

	log.Error().Interface("args", args).
		Msg("No suitable package manager found")
}

// parse out a single installation target for manager from pkg and dispatch
// it to ctx.dirChan
func dPackageBuildDirective(ctx *Context, manager *packageManager, managerName string, pkg Any) {
	dir := &packageDirective{manager: manager, managerName: managerName, cwd: ctx.cwd}

	// RANT oh god!! MY EYES!!! ヽ(ﾟДﾟ)ﾉ
	if pkgStr, ok := pkg.(string); ok {
		if cmdLine, ok := manager.build(manager.execPath, pkgStr, nil); ok {
			dir.cmd = cmdLine
			dir.pkg = pkgStr
			ctx.dirChan <- dir.init(ctx, nil)
		}
		return
	}

	pkgMap, ok := pkg.(map[Any]Any)
	if !ok {
		log.Warn().Msgf("%s targets must be strings or a map containing a %s option, not %T",
			edn.Keyword("package"), edn.Keyword("pkg"), pkg)
		return
	}

	if !directiveMapCondition(ctx, pkgMap) {
		return
	}

	// parse out the name of the package you're installing
	var pkgStr string
	if pkgName, ok := pkgMap[edn.Keyword("pkg")]; !ok {
		log.Warn().Interface("arg", pkg).
			Msgf("Packages maps must specify a %s field", edn.Keyword("pkg"))
		return
	} else if pkgStr, ok = pkgName.(string); !ok {
		log.Warn().Interface("name", pkgName).
			Msgf("Package names must be strings, not %T", pkgName)
		return
	}
	dir.pkg = pkgStr

	if manualCmd, ok := pkgMap[edn.Keyword("manual")]; ok &&
		// manual command specified but we failed to properly generate it so cancel early
		!dShellCommand(ctx, manualCmd, func(manual *shellDirective) { dir.manual = manual }) {
		log.Error().Interface("command", manualCmd).
			Msg("Failed to construct manual command for package installation")
		return
	} else {
		var cmdLine []string
		if cmdLine, ok = manager.build(manager.execPath, pkgStr, pkgMap); !ok {
			return
		}
		dir.cmd = cmdLine
	}

	if before, ok := pkgMap[edn.Keyword("before")]; ok &&
		!dShellCommand(ctx, before, func(before *shellDirective) { dir.before = before }) {
		log.Error().Interface("command", before).
			Msg("Failed to construct before command for package installation")
		return
	}

	if after, ok := pkgMap[edn.Keyword("after")]; ok &&
		!dShellCommand(ctx, after, func(after *shellDirective) { dir.after = after }) {
		log.Error().Interface("command", after).
			Msg("Failed to construct after command for package installation")
		return
	}

	ctx.dirChan <- dir.init(ctx, pkgMap)
}

func (dir *packageDirective) init(ctx *Context, opts map[Any]Any) *packageDirective {
	dir.env = ctx.environ()
	readMapOptionBool(ctx.packageOpts, opts, &dir.interactive, "interactive", false)
	readMapOptionBool(ctx.packageOpts, opts, &dir.stdin, "stdin", dir.interactive)
	readMapOptionBool(ctx.packageOpts, opts, &dir.stdout, "stdour", dir.interactive)
	readMapOptionBool(ctx.packageOpts, opts, &dir.stderr, "stderr", dir.interactive)
	return dir
}

func (dir *packageDirective) run() {
	log.Info().Str("package", dir.pkg).
		Str("manager", dir.managerName).
		Msg("Installing package")

	if dir.manager.sudo && !sudoValidate() {
		return
	}

	if !dir.manager.updated && dir.manager.update != nil {
		updateCmd := dir.manager.update(dir.manager.execPath)
		log.Info().Str("manager", dir.managerName).
			Msg("Updating package archive for manager")
		log.Debug().Strs("cmd", updateCmd).
			Msg("Running subcommand")

		// for now, just inherit all attributes from the package we're installing
		cmd := buildCommand(updateCmd, dir.cwd, dir.env, dir.stdin, dir.stdout, dir.stderr)
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).
				Str("manager", dir.managerName).
				Strs("cmd", updateCmd).
				Msg("Failed to update package archive")
			return
		}
	}

	if dir.before != nil && !dir.before.exec() {
		log.Warn().Msg("Skipping package installation because :before failed")
		return
	}

	ok := true
	if dir.manual != nil {
		log.Debug().Str("package", dir.pkg).
			Msg("Running manual installation command")
		ok = !dir.manual.exec()
	} else {
		cmd := buildCommand(dir.cmd, dir.cwd, dir.env, dir.stdin, dir.stdout, dir.stderr)
		log.Debug().Strs("cmd", dir.cmd).
			Bool("interactive", dir.interactive).
			Msg("Running installation command")
		if err := cmd.Run(); err != nil {
			log.Error().Str("package", dir.pkg).
				Strs("cmd", dir.cmd).
				Err(err).
				Msg("Package installation failed")
		} else {
			log.Debug().Str("package", dir.pkg).
				Msg("Package installed")
		}
	}

	if !ok {
		log.Warn().Str("package", dir.pkg).
			Msg("Failed to install package")
		return
	}

	if dir.after != nil && !dir.after.exec() {
		log.Warn().Msg("Package installation finished but :after failed")
	}
}

func (dir *packageDirective) log() string {
	var res string
	if dir.before != nil {
		res += "package-before " + dir.before.log() + "\n"
	}
	if dir.manual != nil {
		res += "package " + dir.manual.log()
	} else {
		res += "package " + strings.Join(dir.cmd, " ")
	}
	if dir.after != nil {
		res += "package-after " + dir.after.log() + "\n"
	}
	return res
}
