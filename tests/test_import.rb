# frozen_string_literal: true

require 'colorize'
require_relative 'utils'

RSpec.describe :import do
  dotty = Dotty.new

  it 'can import another file' do
    import = 'foo'
    # when trying to import target #{import}, these are all the
    # file paths dotty should accept.
    acceptable_import_paths = [
      "#{import}/dotty.edn",
      "#{import}.dotty",
      "#{import}.edn",
      ".#{import}.edn",
      "#{import}/.config",
      ".#{import}",
      import.to_s
    ].map(&Pathname.method(:new))

    acceptable_import_paths.each do |path|
      dotty.in_config do
        # generate a random message and make the imported file print it out
        msg = rand_str
        path.parent.tap { |p| p.exist? || p.mkdir }
        path.open('w') { |io| io.write("((:debug #{msg.inspect}))") }
        expect(path).to exist

        dotty_run_script "((:import #{import.inspect}))" do |_, _, _, serr|
          # make sure the generated message was printed out
          expect(serr.read.uncolorize).to match(/DBG #{msg}/)
        end
      end
    end
  end

  it 'can supply path through a map' do
    import = 'foo/bar'
    path = Pathname.new('foo/bar')
    msg = rand_str
    dotty.in_config do
      path.parent.tap { |p| p.exist? || p.mkdir }
      path.open('w') { |io| io.write("((:debug #{msg.inspect}))") }
      expect(path).to exist
    end

    dotty_run_script "((:import {:path #{import.inspect}}))" do |_, _, _, serr|
      # make sure the generated message was printed out
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
  end

  it "doesn't import a directory" do
    target = Pathname.new('foo')
    dotty.in_config { target.mkdir }

    dotty.script "((:import #{target.to_path.inspect}))"
    dotty.run_wait do |_, _, serr, proc|
      err = serr.read
      expect(proc.to_i).not_to eq(0), err
      expect(err.uncolorize).to match(/ERR Failed to resolve import target/)
    end
  ensure
    dotty.cleanup
  end
end
