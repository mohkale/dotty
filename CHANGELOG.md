# Changelog
## TODO
* go doc

## [Unreleased]

## [1.0.0] - 2020-09-09
### Added
- Basic directives
  * conditionals (when, and, or, not, bot).
  * shell - execute shell code from configs.
  * mkdir - create directories.
  * clean - remove dead links pointing to dotfiles.
  * link - connect files from src to destination.
  * def - configure defaults values for directives
          and variables to be passed to the shell.
  * log - echo out something through dotty.
  * import - load another config file from this one.
  * package - to install packages using package managers.
- Environment configuration file: .dotty.env.edn, .dotty.env, .dotty
- Install into arbitrary (user specified) home directory
- platform filtering tags
- predicates (:when and :if-bots) for most mapped commands
- only directives, except directives
- prevent cyclic imports

[1.0.0]: https://github.com/mohkale/dotty/releases/tag/1.0.0
