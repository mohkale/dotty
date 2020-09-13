package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"olympos.io/encoding/edn"
)

type packageManager struct {
	// try to install pkg with options
	build func(binPath, pkg string, options map[any]any) ([]string, bool)

	// check for whether this package manager exists
	exists func() string

	// cache so you don't have to keep checking
	execPath string

	// this package manager requires elevated user privileges.
	sudo bool

	update func(binPath string) []string

	// whether the backing archive for this package manager has been
	// updated at least once this session.
	updated bool
}

func findPackageManagerPath(names ...string) string {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			// log.Error().Str("executable", name).
			// 	Str("error", err.Error()).
			// 	Msg("Error when checking for executable")
			return ""
		}

		return path
	}

	return ""
}

// assert whether the current user is an admin or not.
// if not, try to become an admin.
func sudoValidate() bool {
	if isWindows() {
		// windows is weird about sudoers so just leave
		// it till later.
		return true
	}

	cmd := exec.Command("sudo", "--validate")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Error().Int("code", exitErr.ExitCode()).
				Msg("Failed to elevate user privileges")
		} else {
			log.Error().Err(err).
				Msg("Failed to elevate user privileges")
		}
		return false
	}
	return true
}

// The list of package managers dotty can install with.
var packageManagers = map[edn.Keyword]*packageManager{
	// package manager for the python pip module system.
	// See also https://github.com/sobolevn/dotbot-pip
	//
	// Options
	//   global: False
	//   git:
	//     type: github
	//     name: mohkale
	//   # or when you want to use github by default.
	//   git: mohkale
	//
	edn.Keyword("pip"): {
		exists: func() string {
			return findPackageManagerPath("pip", "pip3")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			cmd := []string{binPath, "install"}

			// extract global installation option
			var global bool
			if !readMapOptionBool(nil, opts, &global, "global", false) {
				return nil, false
			}
			if !global {
				cmd = append(cmd, "--user")
			}

			if git, ok := opts[edn.Keyword("git")]; ok {
				var user, host string
				if user, ok = git.(string); ok {
					host = "github"
				} else if gitMap, ok := git.(map[any]any); ok {
					if !readMapOptionString(nil, gitMap, &user, "user", "") ||
						!readMapOptionString(nil, gitMap, &host, "host", "") {
						log.Error().Str("package", pkg).
							Interface("git", git).
							Msg("Missing :user or :host options, skipping package installation")
						return nil, false
					}
				} else {
					log.Warn().Str("package", pkg).
						Interface("git", git).
						Msgf("Pip installation git data must be a string or mapping to :host and :user, not %T", git)
					return nil, false
				}

				pkg = fmt.Sprintf("git+https://%s.com/%s/%s", host, user, pkg)
			}

			return append(cmd, pkg), true
		},
	},

	// Package manager the for golang module system.
	// See also https://github.com/delicb/dotbot-golang
	//
	edn.Keyword("go"): {
		exists: func() string {
			return findPackageManagerPath("go")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			return []string{binPath, "get", pkg}, true
		},
	},

	// Package manager the for cygwins cyg-get package manager
	//
	edn.Keyword("cygwin"): {
		exists: func() string {
			return findPackageManagerPath("cyg-get.bat")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			return []string{binPath, pkg}, true
		},
	},

	// Package manager for rubygems.
	//
	// Options
	//   global: True
	//
	edn.Keyword("gem"): {
		exists: func() string {
			return findPackageManagerPath("gem")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			cmd := []string{binPath, "install"}

			// extract global installation option
			var global bool
			if !readMapOptionBool(nil, opts, &global, "global", false) {
				return nil, false
			}
			if !global {
				cmd = append(cmd, "--user-install")
			}

			return append(cmd, pkg), true
		},
	},

	// Package manager for the chocolatey (windows) package manager.
	// See also https://chocolatey.org/
	edn.Keyword("choco"): {
		exists: func() string {
			return findPackageManagerPath("choco")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			return []string{binPath, "install", "--yes", pkg}, true
		},
	},

	// the pacmans ヽ(^‥^=ゞ)
	edn.Keyword("pacman"): pacmanPackageManager(true, "pacman"),
	edn.Keyword("yay"):    pacmanPackageManager(false, "yay"),
	edn.Keyword("msys"):   pacmanPackageManager(false, "pacman.exe"), // windows doesn't have sudo

	edn.Keyword("apt"): {
		sudo: true,
		exists: func() string {
			return findPackageManagerPath("apt")
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			return []string{"sudo", binPath, "install", "--yes", pkg}, true
		},
		update: func(binPath string) []string {
			return []string{"sudo", binPath, "update"}
		},
	},
}

// For generating pacman like package managers.
//
// because there's sooooo many of them... which is a good thing (●´∀｀●).
// it's just because pacman is that great.
func pacmanPackageManager(sudo bool, filenames ...string) *packageManager {
	return &packageManager{
		sudo: sudo,
		exists: func() string {
			return findPackageManagerPath(filenames...)
		},
		build: func(binPath, pkg string, opts map[any]any) ([]string, bool) {
			cmd := []string{binPath, "-S", "--needed", "--noconfirm", pkg}

			if sudo {
				cmd = append([]string{"sudo"}, cmd...)
			}

			return cmd, true
		},
		update: func(binPath string) []string {
			cmd := []string{binPath, "-Sy"}
			if sudo {
				cmd = append([]string{"sudo"}, cmd...)
			}
			return cmd
		},
	}
}
