require 'colorize'
require_relative 'utils'

RSpec.describe :import do
  it "can import another file" do
    dotty = Dotty.new
    import = "foo"
    # when trying to import target #{import}, these are all the
    # file paths dotty should accept.
    acceptable_import_paths = [
      "#{import}/dotty.edn",
      "#{import}.dotty",
      "#{import}.edn",
      ".#{import}.edn",
      "#{import}/.config",
      ".#{import}",
      "#{import}"
    ].map(&Pathname.method(:new))

    acceptable_import_paths.each do |path|
      dotty.in_config do
        # generate a random message and make the imported file print it out
        msg = rand_str
        path.parent.tap { |p| p.exist? || p.mkdir() }
        path.open("w") { |io| io.write("((:debug #{msg.inspect}))") }
        expect(path).to exist

        dotty_run_script "((:import #{import.inspect}))" do |_, _, _, serr|
          # make sure the generated message was printed out
          expect(serr.read().uncolorize).to match(/DBG #{msg}/)
        end
      end
    end
  end

  it "doesn't import a directory" do
    dotty = Dotty.new
    target = Pathname.new("foo")
    dotty.in_config { target.mkdir() }

    dotty.script "((:import #{target.to_path.inspect}))"
    dotty.run_wait do |_,_,serr,proc|
      err = serr.read()
      expect(proc.to_i).not_to eq(0), err
      expect(err.uncolorize).to match(/ERR failed to resolve import target/)
    end
  ensure
    dotty.cleanup
  end
end