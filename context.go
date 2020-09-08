package main

import (
	"os"

	"github.com/drone/envsubst"
	"github.com/rs/zerolog/log"
)

/**
 * Store for contextual information in the dotty runtime.
 */
type Context struct {
	root string
	cwd  string

	// home directory for the current user.
	// It's kept here because the it shouldn't be modifiable.
	home string

	// The default shell used for subprocesses, this can be overriden.
	// using envOpts.
	shell string

	bots []string

	// send parsed directives through here.
	dirChan chan Directive

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
	return &Context{
		root:        "",
		cwd:         "",
		shell:       "",
		home:        "",
		bots:        make([]string, 0),
		dirChan:     make(chan Directive),
		mkdirOpts:   make(map[string]Any),
		linkOpts:    make(map[string]Any),
		cleanOpts:   make(map[string]Any),
		shellOpts:   make(map[string]Any),
		packageOpts: make(map[string]Any),
		envOpts:     make(map[string]string),
		_env:        nil,
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

func (ctx *Context) fullClone() *Context {
	clone := CreateContext()
	clone.root = ctx.root
	clone.cwd = ctx.cwd
	clone.shell = ctx.shell
	clone.home = ctx.home
	// TODO maybe keep single copy across all instances.
	clone.bots = make([]string, len(ctx.bots))
	copy(clone.bots, ctx.bots)
	clone.dirChan = make(chan Directive)
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
 * An effective clone of the current context.
 *
 * Some fields haven't been cloned because their
 * intended to be shared across all context instances.
 */
func (ctx *Context) clone() *Context {
	c := ctx.fullClone()
	c.dirChan = ctx.dirChan
	return c
}

/**
 * clone the current context and change the cwd.
 */
func (ctx *Context) chdir(cwd string) *Context {
	c := ctx.clone()
	c.cwd = cwd
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
	res, ok := envsubst.Eval(str, ctx.getenv)
	if ok != nil {
		log.Error().Str("str", str).
			Msg("Substitution failed")
		return "", false
	}
	return res, true
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
	for _, installing := range ctx.bots {
		if bot == installing {
			return true
		}
	}
	return false
}
