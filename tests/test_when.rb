# frozen_string_literal: true

require 'colorize'
require_relative './utils'

RSpec.describe :when do
  it 'runs body when condition passes' do
    msg = rand_str
    dotty_run_script "((:when \"true\" (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
  end

  it "doesn't runs body when condition fails" do
    msg = rand_str
    dotty_run_script "((:when \"false\" (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).not_to match(/DBG #{msg}/)
    end
  end

  it 'makes conditions non-interactive by default' do
    msg = rand_str
    dotty_run_script "((:when \"echo #{msg}\" (:debug \"\") ))" do |_dotty, _, sout|
      expect(sout.read.strip).to eq('')
    end
  end

  it 'lets conditions be interactive if desired' do
    msg = rand_str
    dotty_run_script "((:when {:cmd \"echo #{msg}\" :stdout true} (:debug \"\") ))" do |_dotty, _, sout|
      expect(sout.read).to match(/#{msg}/)
    end
  end

  it 'supports negation' do
    msg = rand_str
    dotty_run_script "((:when (:not \"false\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
    dotty_run_script "((:when (:not \"true\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).not_to match(/DBG #{msg}/)
    end
  end

  it 'supports conjunction' do
    msg = rand_str
    dotty_run_script "((:when (:and \"true\" \"true\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
    dotty_run_script "((:when (:and \"true\" \"false\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).not_to match(/DBG #{msg}/)
    end
  end

  it 'supports disjunction' do
    msg = rand_str
    dotty_run_script "((:when (:or \"true\" \"true\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
    dotty_run_script "((:when (:or \"false\" \"false\") (:debug #{msg.inspect}) ))" do |_dotty, _, _, serr|
      expect(serr.read.uncolorize).not_to match(/DBG #{msg}/)
    end
  end

  it 'can run body when installing a bot' do
    msg = rand_str
    dotty = Dotty.new
    dotty_run_script "((:when (:bot \"foobar\") (:debug #{msg.inspect}) ))", dotty, '--bots', 'foobar' do |_, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end
  end

  it 'can skip body when not installing a bot' do
    msg = rand_str
    dotty_run_script "((:when (:bot \"foobar\") (:debug #{msg.inspect}) ))" do |_, _, _, serr|
      expect(serr.read.uncolorize).to_not match(/DBG #{msg}/)
    end
  end

  it 'can accept multiple bots and ANDs them together' do
    msg = rand_str
    dotty = Dotty.new
    dotty_run_script "((:when (:bot \"foobar\" \"bazbag\") (:debug #{msg.inspect}) ))", dotty, '--bots', 'foobar,bazbag' do |_, _, _, serr|
      expect(serr.read.uncolorize).to match(/DBG #{msg}/)
    end

    # when either arg is missing, body is
    %w[foobar bazbag].each do |arg|
      dotty_run_script "((:when (:bot \"foobar\" \"bazbag\") (:debug #{msg.inspect}) ))", dotty, '--bots', arg do |_, _, _, serr|
        expect(serr.read.uncolorize).not_to match(/DBG #{msg}/)
      end
    end
  end
end
