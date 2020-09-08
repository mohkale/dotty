# frozen_string_literal: true

require 'tmpdir'
require 'fileutils'
require 'rbconfig'
require 'open3'

# manager for spawning a dotty test in a temporary directory installing
# into a different temp directory.
class Dotty
  def script(body)
    File.open(File.join(config_dir, 'config.edn'), 'w') do |fd|
      fd.write(body)
    end
  end

  def env(body)
    File.open(File.join(config_dir, '.dotty.env.edn'), 'w') do |fd|
      fd.write(body)
    end
  end

  def run(*flags, &block)
    Open3.popen3(dotty_bin,
                 '--log-level', 'debug',
                 # "--log-json",
                 'install',
                 '--home', install_dir,
                 '--cd', config_dir,
                 *flags, &block)
  end

  def run_wait(*flags, &block)
    run(*flags) do |sin, sout, serr, with_thr|
      block.call(sin, sout, serr, with_thr.value)
    end
  end

  def in_home(&block)
    Dir.chdir(install_dir, &block)
  end

  def in_config(&block)
    Dir.chdir(config_dir, &block)
  end

  def cleanup
    [install_dir, config_dir].each do |dir|
      Dir.each_child(dir) do |file|
        FileUtils.remove_entry_secure(File.join(dir, file), true)
      end
    end
  end

  def install_dir
    self.class.install_dir
  end

  def config_dir
    self.class.config_dir
  end

  def dotty_bin
    self.class.dotty_bin
  end

  def self.install_dir
    @install_dir ||= Dir.mktmpdir('dotty_install')
  end

  def self.config_dir
    @config_dir ||= Dir.mktmpdir('dotty_config')
  end

  def self.dotty_bin
    @dotty_bin = File.join(File.dirname(__dir__), 'dotty') if @dotty_bin.nil?
    @dotty_bin
  end
end

at_exit do
  # clear out temporary directories
  FileUtils.remove_entry_secure(Dotty.install_dir, true)
  FileUtils.remove_entry_secure(Dotty.config_dir, true)
end

# helpers to automate the execution of a dotty config and
# guarantee it exited with the desired exit code.
module DottyHelpers
  # utitlity method to create a new dotty instance, assign
  # a script, run the script, and then run dotty on it. If
  # dotty exits sucesffully, it then passes the inputs of
  # :dotty_run_script: to block.
  def dotty_run_script(script, dotty = nil, *flags, &block)
    dotty = Dotty.new if dotty.nil?
    dotty.script script
    dotty.run_wait(*flags) do |sin, sout, serr, proc|
      err = serr.read
      expect(proc.to_i).to eq(0), err
      block.call(dotty, sin, sout, StringIO.new(err), proc)
    end
  ensure
    dotty.cleanup
  end
end

RSpec.configure do |c|
  c.include DottyHelpers
end

def rand_str(length = 24)
  (0..length).map { rand(65..90).chr }.join
end
