# frozen_string_literal: true

RSpec.describe :def do
  dotty = Dotty.new

  it 'can set environment variables' do
    dotty_run_script '((:def "foo" "bar") (:shell {:cmd "echo foo is $foo" :interactive true}))', dotty do |_, _, sout|
      expect(sout.read.uncolorize).to match(/foo is bar/)
    end
  end

  it 'can reference environment variables in :link' do
    var = 'bar'
    src = Pathname.new('foo')
    dotty.in_config do
      src.open('w')
      expect(src).to exist
    end

    dotty.script "((:def \"foo\" \"~/#{var}\") (:link \"foo\" \"$foo\"))"
    dotty.run_wait do
      dotty.in_home do
        expect(Pathname.new(var)).to exist
      end
    end
  ensure
    dotty.cleanup
  end

  it 'can reference environment variables in :link with #dot/link-gen' do
    var = 'foo'
    src = Pathname.new(var)
    dotty.in_config do
      src.open('w')
      expect(src).to exist
    end

    # TODO: Fix var itself isn't sufficient for #dot/link-gen
    dotty.script "((:def \"foo\" \"$HOME/.#{var}\") #dot/link-gen (:link \"$foo/foo\"))"
    dotty.run_wait do
      dotty.in_home do
        expect(Pathname.new(".#{var}") / var).to exist
      end
    end
  ensure
    dotty.cleanup
  end
end
