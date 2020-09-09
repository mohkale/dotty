<p>
  <a href="https://github.com/mohkale/dotty/blob/master/go.mod">
    <img alt="header" src="./.github/header.jpg"/>
  </a>
</p>

<div align="right">
  <a href="https://github.com/mohkale/dotty">
    <img src="https://img.shields.io/github/go-mod/go-version/mohkale/dotty" />
  </a>
  <a href="https://goreportcard.com/report/github.com/mohkale/dotty">
    <img src="https://goreportcard.com/badge/github.com/mohkale/dotty" />
  </a>
  <a href="https://github.com/mohkale/dotty/actions?query=workflow%3Abuild">
    <img src="https://github.com/mohkale/dotty/workflows/build/badge.svg" />
  </a>
  <a href="https://github.com/mohkale/dotty/actions?query=workflow%3Atests">
    <img src="https://github.com/mohkale/dotty/workflows/tests/badge.svg" />
  </a>
</div>

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Features](#features)
- [Setup](#setup)
- [Example](#example)
- [How it works?](#how-it-works)
- [Directive Format](#directive-format)
    - [File Paths](#file-paths)
- [Directives](#directives)
    - [:mkdir](#mkdir)
    - [:import](#import)
        - [Import Resolution](#import-resolution)
    - [:link](#link)
    - [:clean](#clean)
    - [:shell](#shell)
    - [:def](#def)
    - [:when](#when)
    - [:debug, :info, :warn](#debug-info-warn)
    - [:package](#package)
        - [Package Managers](#package-managers)
            - [:gem](#gem)
            - [:pip](#pip)
- [Tags](#tags)
    - [Link Generation](#link-generation)
    - [Bot Generation](#bot-generation)
- [Using dotty](#using-dotty)
    - [bots](#bots)
    - [.dotty.env](#dottyenv)
- [Credits](#credits)

<!-- markdown-toc end -->

## Features
- modular by design
- gotta go fast, fast, fast
- builtin support for package managers
- (recursive (recursive (recursive (...))))

## Example
Here's a basic config file for setting up python, alongside some python packages. It
should give a high level overview of what your dotfile config will look like using
dotty.

```clojure
(
 #dot/link-gen
 (:link "~/.pdbrc"
        "~/.config/pdbrc.py"
        {:src "pythonrc" :dest "~/.config/pythonrc.py"})

 ;; install python itself
 (:packages (:apt "python3" "python3-pip")
            (:msys "python" "python-pip")
            (:choco "python")
            (:pacman "python3" "python-pip"))

 ;; install the python packages I always want :-)
 (:packages
  (:pip "requests"
        "youtube-dl"
        "beautifulsoup4"
        "edn_format"
        {:pkg "RequestMixin" :git "mohkale"}
        {:pkg "DownloadQueue" :git "mohkale"}))
)
```

## Setup
I recommend creating a script at the root of your dotfiles repository
to automatically fetch dotty when it's not available on your system.

```sh
#!/usr/bin/sh

# exit on any errors
set -e

# which release of dotty you want to use
dotty_url=https://github.com/mohkale/dotty/releases/download/1.0.0/dotty-linux-arm64.tar.gz

# where to put the release
dest_path=./setup/dotty

# download dotty when you don't have it available.
if ! [ -e "$dest_path" ]; then
  curl -o "$dest_path" -L "$dotty_url"
fi

"$dest_path" "$@"
```

If you intend to use dotty across multiple platforms or architectures, you may find
it easier to build dotty from source or to retrieve a dotty version for your current
platform.

By convention each [release][release] contains compiled executables for a majority of
kernels and architectures. You can create a script such as [this][arch] and
[this][kernel] to determine your current architecture and kernel name, and substitute
these into the url above to support cross platform and cross architecture
installations.

[release]: https://github.com/mohkale/dotty
[arch]: https://github.com/mohkale/dotfiles/blob/6454e57d4393d886015a127030894f7d0ca9c3ae/setup/arch
[kernel]: https://github.com/mohkale/dotfiles/blob/6454e57d4393d886015a127030894f7d0ca9c3ae/setup/kernel

```sh
kernel=$(./setup/kernel)
arch=$(./setup/arch)
dotty_url=https://github.com/mohkale/dotty/releases/download/1.0.0/dotty-$kernel-$arch.tar.gz

...
```

Now you create a file at the root of your repository named `config.edn` and dotty
will automatically import it when installing.

```clojure
(
 (:info "hello new dotfiles :grin:")
)
```

## How it works?
dotty reads a configuration file in clojure-style [edn](https://github.com/edn-format/edn)
format which you can scatter across your dotfiles. Each config should be a list
containing a series of *directives* (or actions) that dotty should perform.

For example, this config has a single directive which tells dotty to create some
directories.

```clojure
(
 (:mkdir "~/.config" "~/.local")
)
```

## Directive Format
Most directives support a single argument or extended mapping form. In the single
argument form you simply pass a string or number or basic value, such as:

```clojure
(
 (:mkdir "foo")
)
```

The extended form lets you supply more options to dotty for the argument. Each
directive has it's own definitions as to what options it accepts. The [:mkdir](#mkdir)
directive for example accepts a `:chmod` option to let you set the file permissions
of the directory in octal notation.

```clojure
(
 (:mkdir {:path "foo" :chmod 700})
)
```

Most directives also support two extra options in the mapping form, `:when` and
`:if-bots`.
- `:when` is just a shortcut for the [:when](#when) directive.
- `:if-bots` is a shortcut for `(:when (:bots))`. I.E. `{:if-bots "foo", ...rest}` is
  equivalent to `(:when (:bots "foo") {...rest})`

### File Paths
Most directives let you specify filepaths in a conveniently recursive form. For
example:

```clojure
(
 (:mkdir "foo" ("bar" ("baz" "bag"))))
)
```

creates the directories `foo` and `bar/baz`, `bar/bag`. The syntax is predictable and
can extend upto an arbitrary depth. Paths wrapped into a lower depth are joined into
all possible combinations of earlier paths:

```clojure
(
 (:mkdir ("foo" ("bar" "baz" ("bag") "bam"))
)
```

Will create: `foo/bar/bag` `foo/baz/bag`, `foo/bam`.

You can also substitute environment variables into paths:

```clojure
(
 (:mkdir "${XDG_CONFIG_HOME}/")
)
```

## Directives
### :mkdir
Creates a directory on your file system.

Aliases:
- `:mkdirs`

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :path | Yes | | The path to the directory to create |
| :chmod | | 0744 | Permissions of the newly made directory |

### :import
The `:import` directive lets `dotty` include other config files. This can be chained
with [:when](#when) to conditionally configure dotfiles.

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :path | Yes | | The path to the files to import |

```clojure
(
 ;; you supply one or more paths to configs you want to import
 (:import "foo" "bar" "baz")

 ;; you can conditionally include/exclude a package
 (:import
   {:path "foo"
    :when "[ $HOST -eq foohost ]" })

 ;; you can nest paths using the same syntax as :mkdir
 ;; eg. imports foo/bar and foo/baz
 (:import ("foo" ("bar" "baz")))
)
```

See also [gen-bots](#gen-bots).

#### Import Resolution
`dotty` has a permissive import system relying on little user configuration. For eg.
to import the `foo` config, `dotty` follows the following process:

1. look for a file named `foo.dotty.edn` or `foo.edn` from the current directory.
2. look for a directory called `foo` containing a `dotty.edn` or `foo.edn` or a `.config` file.

### :link
Create a link from one file to another.

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :src | Yes | | The path to the file (or files) that is being linked |
| :dest | Yes | | The path (or paths) where :src is linked to |
| :mkdirs | | true | Automatically create parent directories for :dest |
| :relink | | false | If :dest exists and is a symlink, overwrite it |
| :force | | false | Overwrite :dest if it exists and is not a directory (implies :relink) |
| :glob | | false | :src is a glob path, link all found globs into :dest |
| :ignore-missing | | false | If :src is not found, create a link anyways |
| :symbolic | | true | Whether to create a symlink or a hardlink |

The syntax of the `:link` tag is slightly more peculiar, you specify `:src` then `:dest` in
pairs. If a src is given without a destination, an error is thrown.

```clojure
(
 (:link "./src" "~/dest"
        {:src "foo" :dest "~/bar"})
)
```

You can specify multiple sources or multiple destinations and dotty will handle it
predictably.

```clojure
(
 (:link
   ;; multiple sources, dotty creates a directory at :dest
   ;; and links each source into it.
   ("foo" "bar" "baz") "~/all-my-foos"

   ;; multiple destinations, dotty links src to each :dest
   "foo" ("~/foo1" "~/foo2" "~/foo3")

   ;; multiple sources and destinations, dotty creates a
   ;; directory at each destination and links each src into
   ;; it.
   ("foo" "bar") ("~/foo" "~/bar"))
)
```

If `:dest` exists and is already a directory (or has a trailing slash) dotty will
link the `:src` files into it.

```clojure
(
 (:link
   ;; destination has a trailing slash, make a directory and
   ;; link foo to ~/foo/foo
   "foo" "~/bar/"

   ;; destination exists and is a directory, link baz into it.
   "baz" "~/.config")
)
```

See also [link-gen](link-gen).

### :clean
Finds and remove any broken links that point to your dotfiles. The format is the same
as [:mkdir](#mkdir)

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :path | yes | | The path to the directories to clean |
| :recursive | | false | Recursively search for dead links |
| :force | | false | Remove broken links even if they don't point to dotfiles |

```clojure
(
 (:clean {:path "~/.local/bin" :recursive true})
)
```

### :shell
Lets you execute arbitrary shell code.

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :cmd | yes | | The shell command to run |
| :desc | | | A brief description of what this command does (for logging) |
| :quiet | | false | Don't log the command, or it's exit code |
| :stdin | | :interactive | Connect this commands :stdin to dottys stdin |
| :stdout | | :interactive | Connect this commands :stdout to dottys stdout |
| :stderr | | :interactive | Connect this commands :stderr to dottys stderr |
| :interactive | | false | Set the defaults for :stdin, :stdout, :stderr |

Each argument passed to this directive is executed in its own subshell. To run
multiple commands in the same subshell, pass it as a single (long string) or as
multiple entries in a list.

```clojure
(
 ;; doesn't output anything because :stdout is false and each argument
 ;; is run in a different subshell.
 (:shell "foo=hello" "echo $foo")

 ;; doesn't output anything because :stdout is false
 (:shell ("foo=hello" "echo $foo"))
 (:shell "foo=hello
          echo $foo")

 ;; outputs correctly
 (:shell {:desc "Prints foo"
          :stdout true
          :cmd "foo=hello
                echo $foo"})
)
```

### :def
Assign default options for directives or environment variables.

This directive doesn't support an extended map form. It accepts only key value pairs,
with default behaviour being assigning each key to it's value in the environment
variables exposed to subprocesses.

```clojure
(
 (:def
   ;; export foo=bar and "bag=bag" as environment variables
   "foo" "bar"
   "baz" "bag"

   ;; set the default value of some options for the :shell directive.
   (:shell :quiet       false
           :interactive true))

 ;; outputs without issue
 (:shell "echo $foo; echo $baz")
)
```

Directives that support the `:def` directive are:
- `:mkdir`
- `:link`
- `:clean`
- `:shell`
- `:package`

### :when
Conditionally execute some directives.

This directive lets you use `:shell` to conditionally perform tasks. It takes the
same arguments as the `:shell` directive and supports chaining and conditionals using
`:and`, `:not` and `:or`.

```clojure
(
 (:when (:not "uname -a | grep linux")
   (:warn "You're not in linux, why?"))
)
```

The when directive changes the defaults of `:shell` to make `:quiet` true by default.
This is because conditionals are expected to pass or fail, it's not an error if they
fail. You can override this change if you prefer:

```clojure
(
 ;; an error log is thrown if this command fails
 (:when {:cmd "uname -a | grep linux" :quiet false}
   (:warn "You're in linux, Yippee"))
)
```

### :debug, :info, :warn
These directives let you hook into dottys logger to produce your own logging output.

```clojure
(
 (:debug "hello world")
 ;; supports printf style formatting
 (:info "my name is: %s" "mohkale")
 (:warn "you're environment isn't setup correctly")
)
```

### :package
Use an external package manager to install a package.

Aliases:
- `:packages`

The format of this directive resembles a switch statement, dotty will try each
package manager in turn until it finds one that exists and then it'll try to install
all supplied packages with that manager.

```clojure
(
 (:packages
   ;; if pip is available, try to install foo and bar with it.
   (:pip "foo" "bar")

   ;; otherwise if rubygems is available, try to install these gems.
   (:gem "baz" "bag")

   ;; the :default clause runs a shell command when no managers were
   ;; found.
   (:default "echo failed to find a package manager, （；_・）")
   )
)
```

Each package supports the extended map form with the following options:

| Option  | Is Default | Default Value | Description |
|---|---|---|---|
| :pkg | yes | | The package to install |
| :manual | | | A shell command to use to install the package instead of the manager. |
| :before | | | A shell command to run before installing the package. Installation is skipped if this fails |
| :after | | | A shell command to run after installing the package. |
| :stdin | | :interactive | Connect the package install commands :stdin to dottys stdin |
| :stdout | | :interactive | Connect the package install commands :stdout to dottys stdout |
| :stderr | | :interactive | Connect the package install commands :stderr to dottys stderr |
| :interactive | | true | Set the defaults for :stdin, :stdout, :stderr |

For example, here's a config for installing nodejs:

```clojure
(
 (:packages
  (:apt {:pkg "nodejs"
         :before "[ -z \"$(which node)\" ] || exit 0 # already installed
                  curl -sL https://deb.nodesource.com/setup_13.x | sudo bash -"})
  (:choco "nodejs")
  (:pacman "nodejs"))
)
```

#### Package Managers
I've only ever used windows & ubuntu/arch so the only package managers I have
configured is for them. If you'd like to add support for a newer package managers,
simply navigate to [d_pacmans.go](./d_pacmans.go) and add a new struct to the
`packageManagers` map.

dotty currently supports:
- [:pip](https://pypi.org/project/pip/)
- [:go](https://golang.org/)
- [:cygwin](https://www.cygwin.com/)
- [:gem](https://rubygems.org/)
- [:choco](https://chocolatey.org/)
- [:pacman](https://wiki.archlinux.org/index.php/pacman)
- [:yay](https://github.com/Jguer/yay)
- [:msys](https://www.msys2.org/)
- [:apt](https://en.wikipedia.org/wiki/APT_(software))

**WARN**: Some of these package managers may require elevated user privilages. When
possible dotty will ask for sudo privilages automatically.

And the following managers support extended options.

##### :gem
| Option  | Default Value | Description |
|---|---|---|
| :global | false | Whether to install this gem globally |

##### :pip
| Option  | Default Value | Description |
|---|---|---|
| :global | false | Whether to install this python package globally |
| :git | | Install from git instead of PyPI |

NOTE: the `:git` option has two forms, you can supply it as as a single string, in
which case dotty will assume a default host of github. Or you can supply it as
a map with both the git host and user name for where the package can be found.

```clojure
;; both of these forms install: git+https://github.com/mohkale/foo
{:pkg "foo" :git "mohkale"}
{:pkg "foo" :git {:host "github" :user "mohkale"}}
```

## Tags
Tags are a good way to preprocess some data before interpreting it. It's an [edn
construct][edn-tags].

[edn-tags]: https://github.com/edn-format/edn#tagged-elements

Dotty offers the following tags to simplify configurations.

| Tag               | Affect                                                   |
|-------------------|----------------------------------------------------------|
| #dot/only-windows | Only run this directive when on Microsoft windows.       |
| #dot/only-linux   | Only run this directive when on a Linux system.          |
| #dot/only-darwin  | Only run this directive when on a MacOS system.          |
| #dot/only-unix    | Only run this directive when on a Linux or MacOS system. |
| #dot/link-gen     | Automatically generate link srcs (or destinations).      |
| #dot/gen-bots     | Automatically generate `:if-bots` options for imports.   |

### Link Generation
Quite often when you're linking files the destination matches the source file (likely
without a leading '.'). To avoid having to repeat the same name multiple times, you
can attach the `#dot/link-gen` tag to a link directive and dotty will try guess the
src from your destinations.

```clojure
(
 #dot/link-gen
 (:link
  "~/.bashrc"                            ; :src is bashrc
  {:src "bash_logout"}                   ; :dest is ~/.bash_logout

  ;; when both are provided, dotty leaves them alone.
  {:src "profile" :dest "~/.bash_profile"})
)
```

### Bot Generation
Quite often you're likely to end up with multiple sub configs for different programs
that you want to let users choose to setup. To do so you'd have to specify a
`:if-bots` options for every import.

```clojure
(
 (:import
   {:path "lf"            :if-bots "lf"}
   {:path "ranger"        :if-bots "ranger"}
   {:path "langs/python"  :if-bots "python"}
   {:path "langs/node"    :if-bots "node"})
)
```

This quickly becomes a mess. The `#dot/gen-bots` tag automatically inserts a `:if-bots`
option for each entry in an import directive (unless one is already found).

```clojure
(
 #dot/gen-bots
 (:import
   "lf"
   "ranger"
   "langs/python"
   "langs/node")
)
```

This is equivalent to the previous configuration (and I think unquestionably nicer to
read).

## Using dotty
### bots
As someone who jumps between different platforms as a creature of habit, I've grown
accustomed to only configuring what I end up using. dotty is designed to make this
easier by letting you specify (at dotfile installation) what you want installed? or
what you have available to install.

This simple mechanism is called **bots**. You specify which bots you want to install
when you invoke dotty (using the `-b` flag). For example, if I want to install the
python and ruby bot I can pass:

```sh
dotty install -b python,ruby
```

Now in our configuration we can use the [:when](#when) directive to conditionally
configure logic when we're installing these bots:

```clojure
(
 (:when (:bot "python")
   (:info "Installing python"))
)
```

Better yet, we can modularise our config into a subconfig and only import it when
we're installing python:

```clojure
(
 (:import {:path "langs/python" :if-bots "python"})
)
```

Once dotty is finished installing your dotfiles, it'll append the bots you just
installed into a file local to your dotfiles (see `dotty install -h`) which you
can use to resync them at a later date.

```sh
dotty install --only link -b "$(cat .dotty.bots)"
```

This will run any link directives for any bots we've installed in the past and save
to `.dotty.bots`. NOTE: you can override the default file name/path for the bots file
using the `DOTTY_BOTS_FILE` environment variable.

dotty can also traverse your dotfiles and list any bots you're checking for at any
stage. This can let you see what bots your dotfiles have available. For more
information, see `dotty list-bots`.

### .dotty.env
At startup dotty can read environment configurations from a file at one of
`.dotty.env.edn`, `.dotty.env`, `.dotty`. This file is a essentially a dotty
configuration automatically wrapped in a [:def](#def) tag. For example:

```clojure
(
 ;; specify environment variables
 "XDG_CONFIG_HOME" "~/.config"

 ;; change directive defaults
 (:shell :interactive true)
)
```

## Credits
`dotty` takes more than a little inspiration from [dotbot][dbot], the dotfile management
solution I was using before creating this. Give that project some love if you can :heart:.

[dbot]: https://github.com/anishathalye/dotbot
