package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

// Context - store for contextual information in the dotty runtime.
type Context struct {
	Root string
	Cwd  string

	// home directory for the current user.
	// It's kept here because the it shouldn't be modifiable.
	Home string

	// The default shell used for subprocesses, this can be overridden.
	// using envOpts.
	Shell string

	// The path to all the configs we've imported.
	imports *[]string

	Bots []string

	OnlyDirectives   []string
	ExceptDirectives []string

	// send parsed directives through here.
	DirChan chan directive

	// Key/Value options for specific directives or subshell environments.
	mkdirOpts   map[string]Any
	linkOpts    map[string]Any
	cleanOpts   map[string]Any
	shellOpts   map[string]Any
	packageOpts map[string]Any
	envOpts     map[string]string

	// generated environment of the form that exec.Command can accept.
	_env []string
}

func CreateContext() *Context {
	imports := make([]string, 0)
	return &Context{
		Root:             "",
		Cwd:              "",
		Shell:            "",
		Home:             "",
		Bots:             make([]string, 0),
		imports:          &imports,
		DirChan:          make(chan directive),
		mkdirOpts:        make(map[string]Any),
		linkOpts:         make(map[string]Any),
		cleanOpts:        make(map[string]Any),
		OnlyDirectives:   make([]string, 0),
		ExceptDirectives: make([]string, 0),
		shellOpts:        make(map[string]Any),
		packageOpts:      make(map[string]Any),
		envOpts:          make(map[string]string),
		_env:             nil,
	}
}

/**
 * Get options map for the directive associated with key.
 */
func (ctx *Context) optsFromString(key string) (map[string]Any, bool) {
	switch {
	case key == "mkdirs":
		fallthrough
	case key == "mkdir":
		return ctx.mkdirOpts, true
	case key == "link":
		return ctx.linkOpts, true
	case key == "clean":
		return ctx.cleanOpts, true
	case key == "shell":
		return ctx.shellOpts, true
	case key == "packages":
		fallthrough
	case key == "package":
		return ctx.packageOpts, true
	}

	return nil, false
}

func _cloneDirectiveOpts(src map[string]Any, dest map[string]Any) {
	for key, value := range src {
		dest[key] = value
	}
}

/**
 * An effective clone of the current context.
 *
 * Some fields haven't been cloned because their
 * intended to be shared across all context instances.
 */
func (ctx *Context) clone() *Context {
	clone := CreateContext()
	// basic types so they're auto immutable
	clone.Root = ctx.Root
	clone.Cwd = ctx.Cwd
	clone.Shell = ctx.Shell
	clone.Home = ctx.Home

	// fields that should be shared across all instances
	// NOTE These aren't modifiable.
	clone.Bots = ctx.Bots
	clone.DirChan = ctx.DirChan
	clone.OnlyDirectives = ctx.OnlyDirectives
	clone.ExceptDirectives = ctx.ExceptDirectives
	clone.imports = ctx.imports

	// Fields that are expected to be mutated at different points.
	_cloneDirectiveOpts(ctx.mkdirOpts, clone.mkdirOpts)
	_cloneDirectiveOpts(ctx.linkOpts, clone.linkOpts)
	_cloneDirectiveOpts(ctx.cleanOpts, clone.cleanOpts)
	_cloneDirectiveOpts(ctx.shellOpts, clone.shellOpts)
	_cloneDirectiveOpts(ctx.packageOpts, clone.packageOpts)
	for key, value := range ctx.envOpts {
		clone.envOpts[key] = value
	}

	return clone
}

/**
 * clone the current context and change the cwd.
 */
func (ctx *Context) chdir(cwd string) *Context {
	c := ctx.clone()
	c.Cwd = cwd
	return c
}

/**
 * substitute variables from the current context environment
 * into str. Without first building an entire environment map.
 */
func (ctx *Context) getenv(str string) string {
	if val, ok := ctx.envOpts[str]; ok {
		return val
	}

	if val, ok := os.LookupEnv(str); ok {
		return val
	}

	log.Warn().Str("var", str).
		Msg("Failed to find environment variable")

	return ""
}

func (ctx *Context) eval(str string) (string, bool) {
	return os.Expand(str, ctx.getenv), true
}

/**
 * context environment has been modified, environ() needs to be rebuilt.
 */
func (ctx *Context) invalidateEnv() {
	ctx._env = nil
}

func (ctx *Context) environ() []string {
	if ctx._env == nil {
		env := os.Environ()
		ctx._env = make([]string, len(env)+len(ctx.cleanOpts)+
			len(ctx.envOpts)+len(ctx.linkOpts)+len(ctx.mkdirOpts)+
			len(ctx.shellOpts))
		i := copy(ctx._env, env)

		for key, value := range ctx.envOpts {
			ctx._env[i] = key + "=" + value
			i++
		}
	}

	return ctx._env
}

func (ctx *Context) installingBot(bot string) bool {
	// TODO optomize, linear search bad :P
	for _, installing := range ctx.Bots {
		if bot == installing {
			return true
		}
	}
	return false
}

func (ctx *Context) skipDirectivePredicate(dir string) bool {
	if len(ctx.ExceptDirectives) > 0 &&
		StringSliceContains(ctx.ExceptDirectives, dir) {
		return true
	} else if len(ctx.OnlyDirectives) > 0 &&
		!StringSliceContains(ctx.OnlyDirectives, dir) {
		return true
	}

	return false
}
